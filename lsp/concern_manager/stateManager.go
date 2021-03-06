package concern_manager

import (
	"errors"
	miraiBot "github.com/Logiase/MiraiGo-Template/bot"
	"github.com/Logiase/MiraiGo-Template/config"
	"github.com/Logiase/MiraiGo-Template/utils"
	"github.com/Sora233/DDBOT/concern"
	localdb "github.com/Sora233/DDBOT/lsp/buntdb"
	localutils "github.com/Sora233/DDBOT/utils"
	"github.com/Sora233/sliceutil"
	"github.com/tidwall/buntdb"
	"strings"
	"time"
)

var logger = utils.GetModuleLogger("concern_manager")
var ErrEmitQNotInit = errors.New("emit queue not enabled")

type StateManager struct {
	*localdb.ShortCut
	KeySet
	emitChan  chan *localutils.EmitE
	emitQueue *localutils.EmitQueue
	useEmit   bool
}

// GetGroupConcernConfig always return non-nil
func (c *StateManager) GetGroupConcernConfig(groupCode int64, id interface{}) (concernConfig *GroupConcernConfig) {
	var err error
	err = c.RTxCover(func(tx *buntdb.Tx) error {
		val, err := tx.Get(c.GroupConcernConfigKey(groupCode, id))
		if err != nil {
			return err
		}
		concernConfig, err = NewGroupConcernConfigFromString(val)
		return err
	})
	if err != nil && err != buntdb.ErrNotFound {
		logger.WithFields(localutils.GroupLogFields(groupCode)).WithField("id", id).Errorf("GetGroupConcernConfig error %v", err)
	}
	if concernConfig == nil {
		concernConfig = new(GroupConcernConfig)
	}
	return
}

// OperateGroupConcernConfig 在一个rw事务中获取GroupConcernConfig并交给函数，如果返回true，就保存GroupConcernConfig，否则就回滚。
func (c *StateManager) OperateGroupConcernConfig(groupCode int64, id interface{}, f func(concernConfig *GroupConcernConfig) bool) error {
	err := c.RWTxCover(func(tx *buntdb.Tx) error {
		var concernConfig *GroupConcernConfig
		var configKey = c.GroupConcernConfigKey(groupCode, id)
		val, err := tx.Get(configKey)
		if err == nil {
			concernConfig, err = NewGroupConcernConfigFromString(val)
		} else if err == buntdb.ErrNotFound {
			concernConfig = new(GroupConcernConfig)
			err = nil
		}
		if err != nil {
			return err
		}
		if !f(concernConfig) {
			return localdb.ErrRollback
		}
		_, _, err = tx.Set(configKey, concernConfig.ToString(), nil)
		return err
	})
	return err
}

// CheckAndSetAtAllMark 检查@全体标记是否过期，已过期返回true，并重置标记，否则返回false。
// 因为@全体有次数限制，并且较为恼人，故设置标记，两次@全体之间必须有间隔。
func (c *StateManager) CheckAndSetAtAllMark(groupCode int64, id interface{}) (result bool) {
	_ = c.RWTxCover(func(tx *buntdb.Tx) error {
		key := c.GroupAtAllMarkKey(groupCode, id)
		_, err := tx.Get(key)
		if err == buntdb.ErrNotFound {
			result = true
			_, _, err = tx.Set(key, "", localdb.ExpireOption(time.Hour*2))
			if err != nil {
				// 如果设置失败，可能会造成连续at
				result = false
			}
		}
		return nil
	})
	return
}

func (c *StateManager) CheckGroupConcern(groupCode int64, id interface{}, ctype concern.Type) error {
	return c.RTxCover(func(tx *buntdb.Tx) error {
		val, err := tx.Get(c.GroupConcernStateKey(groupCode, id))
		if err == nil {
			if concern.FromString(val).ContainAll(ctype) {
				return ErrAlreadyExists
			}
		}
		return nil
	})
}

func (c *StateManager) CheckConcern(id interface{}, ctype concern.Type) error {
	state, err := c.GetConcern(id)
	if err != nil {
		return err
	}
	if state.ContainAll(ctype) {
		return ErrAlreadyExists
	}
	return nil
}

func (c *StateManager) AddGroupConcern(groupCode int64, id interface{}, ctype concern.Type) (newCtype concern.Type, err error) {
	groupStateKey := c.GroupConcernStateKey(groupCode, id)
	oldCtype, err := c.GetConcern(id)
	if err != nil {
		return concern.Empty, err
	}
	newCtype, err = c.upsertConcernType(groupStateKey, ctype)
	if err != nil {
		return concern.Empty, err
	}

	if c.useEmit && oldCtype.Empty() {
		for _, t := range ctype.Split() {
			c.emitQueue.Add(localutils.NewEmitE(id, t), time.Time{})
		}
	}
	return
}

func (c *StateManager) RemoveGroupConcern(groupCode int64, id interface{}, ctype concern.Type) (newCtype concern.Type, err error) {
	err = c.RWTxCover(func(tx *buntdb.Tx) error {
		groupStateKey := c.GroupConcernStateKey(groupCode, id)
		val, err := tx.Get(groupStateKey)
		if err != nil {
			return err
		}
		oldState := concern.FromString(val)
		newCtype = oldState.Remove(ctype)
		if newCtype == concern.Empty {
			_, err = tx.Delete(groupStateKey)
		} else {
			_, _, err = tx.Set(groupStateKey, newCtype.String(), nil)
		}
		if err != nil {
			return err
		}
		return nil
	})
	return
}

func (c *StateManager) RemoveAllByGroupCode(groupCode int64) (err error) {
	return c.RWTxCover(func(tx *buntdb.Tx) error {
		var removeKey []string
		var iterErr error
		var indexes = []string{
			c.GroupConcernStateKey(groupCode),
			c.GroupConcernConfigKey(groupCode),
		}
		for _, key := range indexes {
			iterErr = tx.Ascend(key, func(key, value string) bool {
				removeKey = append(removeKey, key)
				return true
			})
			if iterErr != nil {
				return iterErr
			}
		}
		for _, key := range removeKey {
			tx.Delete(key)
		}
		for _, key := range indexes {
			tx.DropIndex(key)
		}
		return nil
	})
}

func (c *StateManager) RemoveAllById(_id interface{}) (err error) {
	return c.RWTxCover(func(tx *buntdb.Tx) error {
		var removeKey []string
		var iterErr error
		iterErr = tx.Ascend(c.GroupConcernStateKey(), func(key, value string) bool {
			var id interface{}
			_, id, iterErr = c.ParseGroupConcernStateKey(key)
			if id == _id {
				removeKey = append(removeKey, key)
			}
			return true
		})
		if iterErr != nil {
			return iterErr
		}
		for _, key := range removeKey {
			tx.Delete(key)
		}
		return nil
	})
}

// GetGroupConcern return the concern.Type in specific group for a id
func (c *StateManager) GetGroupConcern(groupCode int64, id interface{}) (result concern.Type, err error) {
	err = c.RTxCover(func(tx *buntdb.Tx) error {
		val, err := tx.Get(c.GroupConcernStateKey(groupCode, id))
		if err != nil {
			return err
		}
		result = concern.FromString(val)
		return nil
	})
	return result, err
}

// GetConcern return the concern.Type combined from all group for a id
func (c *StateManager) GetConcern(id interface{}) (result concern.Type, err error) {
	_, _, _, err = c.List(func(groupCode int64, _id interface{}, p concern.Type) bool {
		if id == _id {
			result = result.Add(p)
		}
		return true
	})
	return
}

func (c *StateManager) List(filter func(groupCode int64, id interface{}, p concern.Type) bool) (idGroups []int64, ids []interface{}, idTypes []concern.Type, err error) {
	err = c.RTxCover(func(tx *buntdb.Tx) error {
		var iterErr error
		err := tx.Ascend(c.GroupConcernStateKey(), func(key, value string) bool {
			var groupCode int64
			var id interface{}
			groupCode, id, iterErr = c.ParseGroupConcernStateKey(key)
			if iterErr != nil {
				return false
			}
			ctype := concern.FromString(value)
			if ctype == concern.Empty {
				return true
			}
			if filter(groupCode, id, ctype) == true {
				idGroups = append(idGroups, groupCode)
				ids = append(ids, id)
				idTypes = append(idTypes, ctype)
			}
			return true
		})
		if err != nil {
			return err
		}
		if iterErr != nil {
			return iterErr
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return
}

func (c *StateManager) ListByGroup(groupCode int64, filter func(id interface{}, p concern.Type) bool) (ids []interface{}, idTypes []concern.Type, err error) {
	err = c.RTxCover(func(tx *buntdb.Tx) error {
		var iterErr error
		err := tx.Ascend(c.GroupConcernStateKey(groupCode), func(key, value string) bool {
			var id interface{}
			_, id, iterErr = c.ParseGroupConcernStateKey(key)
			if iterErr != nil {
				return false
			}
			ctype := concern.FromString(value)
			if filter == nil || filter(id, ctype) == true {
				ids = append(ids, id)
				idTypes = append(idTypes, ctype)
			}
			return true
		})
		if err != nil {
			return err
		}
		if iterErr != nil {
			return iterErr
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return
}

func (c *StateManager) ListIds() (ids []interface{}, err error) {
	var idSet = make(map[interface{}]bool)
	_, _, _, err = c.List(func(groupCode int64, id interface{}, p concern.Type) bool {
		idSet[id] = true
		return true
	})
	if err != nil {
		return nil, err
	}
	for k := range idSet {
		ids = append(ids, k)
	}
	return ids, nil
}

func (c *StateManager) GroupTypeById(ids []interface{}, types []concern.Type) ([]interface{}, []concern.Type, error) {
	if len(ids) != len(types) {
		return nil, nil, ErrLengthMismatch
	}
	var (
		idTypeMap  = make(map[interface{}]concern.Type)
		result     []interface{}
		resultType []concern.Type
	)
	for index := range ids {
		id := ids[index]
		t := types[index]
		if old, found := idTypeMap[id]; found {
			idTypeMap[id] = old.Add(t)
		} else {
			idTypeMap[id] = t
		}
	}

	for id, t := range idTypeMap {
		result = append(result, id)
		resultType = append(resultType, t)
	}
	return result, resultType, nil
}

func (c *StateManager) FreshCheck(id interface{}, setTTL bool) (result bool, err error) {
	err = c.RWTxCover(func(tx *buntdb.Tx) error {
		freshKey := c.FreshKey(id)
		_, err := tx.Get(freshKey)
		if err == buntdb.ErrNotFound {
			result = true
			if setTTL {
				_, _, err = tx.Set(freshKey, "", localdb.ExpireOption(time.Minute))
				if err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		}
		return nil
	})
	return
}

func (c *StateManager) FreshIndex(groups ...int64) {
	db, err := localdb.GetClient()
	if err != nil {
		return
	}
	db.CreateIndex(c.GroupConcernStateKey(), c.GroupConcernStateKey("*"), buntdb.IndexString)
	db.CreateIndex(c.GroupConcernConfigKey(), c.GroupConcernConfigKey("*"), buntdb.IndexString)
	db.CreateIndex(c.GroupAtAllMarkKey(), c.GroupAtAllMarkKey("*"), buntdb.IndexString)
	db.CreateIndex(c.FreshKey(), c.FreshKey("*"), buntdb.IndexString)
	if len(groups) == 0 {
		for _, groupInfo := range miraiBot.Instance.GroupList {
			db.CreateIndex(c.GroupConcernStateKey(groupInfo.Code), c.GroupConcernStateKey(groupInfo.Code, "*"), buntdb.IndexString)
			db.CreateIndex(c.GroupConcernConfigKey(groupInfo.Code), c.GroupConcernConfigKey(groupInfo.Code, "*"), buntdb.IndexString)
			db.CreateIndex(c.GroupAtAllMarkKey(groupInfo.Code), c.GroupAtAllMarkKey(groupInfo.Code, "*"), buntdb.IndexString)
			db.CreateIndex(c.FreshKey(groupInfo.Code), c.FreshKey(groupInfo.Code, "*"), buntdb.IndexString)
		}
	} else {
		for _, g := range groups {
			db.CreateIndex(c.GroupConcernStateKey(g), c.GroupConcernStateKey(g, "*"), buntdb.IndexString)
			db.CreateIndex(c.GroupConcernConfigKey(g), c.GroupConcernConfigKey(g, "*"), buntdb.IndexString)
			db.CreateIndex(c.GroupAtAllMarkKey(g), c.GroupAtAllMarkKey(g, "*"), buntdb.IndexString)
			db.CreateIndex(c.FreshKey(g), c.FreshKey(g, "*"), buntdb.IndexString)
		}
	}
}

func (c *StateManager) FreshAll() {
	miraiBot.Instance.ReloadFriendList()
	miraiBot.Instance.ReloadGroupList()
	db, err := localdb.GetClient()
	if err != nil {
		return
	}
	allIndex, err := db.Indexes()
	if err != nil {
		return
	}
	for _, index := range allIndex {
		if strings.HasPrefix(index, c.GroupConcernStateKey()+":") {
			db.DropIndex(index)
		}
	}

	c.FreshIndex()

	var groupCodes []int64
	for _, groupInfo := range miraiBot.Instance.GroupList {
		groupCodes = append(groupCodes, groupInfo.Code)
	}
	var removeKey []string

	c.RTxCover(func(tx *buntdb.Tx) error {
		tx.Ascend(c.GroupConcernStateKey(), func(key, value string) bool {
			groupCode, _, err := c.ParseGroupConcernStateKey(key)
			if err != nil {
				removeKey = append(removeKey, key)
			} else if !sliceutil.Contains(groupCodes, groupCode) {
				removeKey = append(removeKey, key)
			}
			return true
		})
		return nil
	})
	c.RWTxCover(func(tx *buntdb.Tx) error {
		for _, key := range removeKey {
			tx.Delete(key)
		}
		return nil
	})
}

func (c *StateManager) upsertConcernType(key string, ctype concern.Type) (newCtype concern.Type, err error) {
	err = c.RWTxCover(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err == buntdb.ErrNotFound {
			newCtype = ctype
			tx.Set(key, ctype.String(), nil)
		} else if err == nil {
			newCtype = concern.FromString(val).Add(ctype)
			tx.Set(key, newCtype.String(), nil)
		} else {
			return err
		}
		return nil
	})
	return
}

func (c *StateManager) Start() error {
	if c.useEmit {
		_, ids, types, err := c.List(func(groupCode int64, id interface{}, p concern.Type) bool {
			return true
		})
		if err != nil {
			return err
		}
		ids, types, err = c.GroupTypeById(ids, types)
		if err != nil {
			return err
		}
		for index := range ids {
			for _, t := range types[index].Split() {
				c.emitQueue.Add(localutils.NewEmitE(ids[index], t), time.Now())
			}
		}
	}
	return nil
}

func (c *StateManager) EmitFreshCore(name string, fresher func(ctype concern.Type, id interface{}) error) {
	if !c.useEmit {
		return
	}
	for e := range c.emitChan {
		id := e.Id
		if ok, _ := c.FreshCheck(id, true); !ok {
			logger.WithField("id", id).WithField("result", ok).Trace("fresh check failed")
			continue
		}
		logger.WithField("id", id).Trace("fresh")
		if err := fresher(e.Type, id); err != nil {
			logger.WithField("id", id).WithField("name", name).Errorf("fresher error %v", err)
		}
	}
}

func NewStateManager(keySet KeySet, useEmit bool) *StateManager {
	sm := &StateManager{
		emitChan: make(chan *localutils.EmitE),
		KeySet:   keySet,
		useEmit:  useEmit,
	}
	if useEmit {
		var interval time.Duration
		if config.GlobalConfig == nil {
			interval = time.Second * 5
		} else {
			interval = config.GlobalConfig.GetDuration("concern.emitInterval")
		}
		sm.emitQueue = localutils.NewEmitQueue(sm.emitChan, interval)
	}
	return sm
}
