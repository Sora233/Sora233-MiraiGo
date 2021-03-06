# DDBOT

[<img src="https://github.com/Sora233/DDBOT/actions/workflows/ci.yml/badge.svg"/>](https://github.com/Sora233/DDBOT/actions/workflows/ci.yml)
[<img src="https://coveralls.io/repos/github/Sora233/DDBOT/badge.svg?branch=master"/>](https://coveralls.io/github/Sora233/DDBOT)

一个基于MiraiGO的多功能QQ群机器人。

[Bilibili专栏](https://www.bilibili.com/read/cv10602230)

-----

## 设计理念

制作bot的本意是为了减轻一些重复的工作负担，bot只会做好bot份内的工作：

- ddbot的交互被刻意设计成最小程度，正常交流时永远不必担心会误触ddbot。
- ddbot只有两种情况会主动发言，更新动态和直播，以及答复命令结果。

## **基本功能：**

- **B站直播/动态推送**
  - 让阁下在DD的时候不错过任何一场突击。
- **斗鱼直播推送**
  - 没什么用，主要用来看爽哥。
- **油管直播/视频推送**
  - 支持推送预约直播信息及视频更新。
- **虎牙直播推送** *新增*
  - 不知道能看谁。
- 可配置的 **@全体成员** *新增*
  - 只建议单推群开启。
- **人脸识别**
  - 主要用来玩，支持二次元人脸。
- **倒放**
  - 主要用来玩。
- **Roll**
  - 没什么用的roll点。
- **签到**
  - 没什么用的签到。
- **权限管理**
  - 可配置整个命令的启用和禁用，也可对单个用户配置命令权限，防止滥用。
- **帮助**
  - 输出一些没什么帮助的信息。

<details>
  <summary>里命令</summary>

以下命令默认禁用，使用enable命令后才能使用

- **随机图片**
    - 由 [api.olicon.app](https://api.lolicon.app/#/) 提供
- **色图判定**
    - 由阿里云提供
    - **注意：阿里云该服务2021年3月25日开始收费**

</details>

### 推送效果

<img src="https://user-images.githubusercontent.com/11474360/111737379-78fbe200-88ba-11eb-9e7e-ecc9f2440dd8.jpg" width="300">

### 用法示例

详细介绍及示例请查看：[详细示例](/EXAMPLE.md)

阁下可添加Demo机器人体验，QQ号：

- 368236249 （二号机）
- 1561991863 （初号机）

<details>
<summary>点此扫码二号机</summary>
<img src="https://user-images.githubusercontent.com/11474360/122684719-a8afe280-d239-11eb-9089-b8ce6613c819.jpg" width="300" height="450">
</details>

<details>
<summary>点此扫码初号机</summary>
<img src="https://user-images.githubusercontent.com/11474360/108590360-150afa00-739e-11eb-86f7-77f68d845505.jpeg" width="300" height="450">
</details>

推荐您优先选择二号机，由于目前初号机负载较高。

**尝试同时使用多个Demo机器人会导致您被暂时加入黑名单**

## 使用与部署

对于普通用户，推荐您选择使用开放的Demo机器人。

如果您需要私人部署，[详见部署指南](/INSTALL.md)

如果您遇到任何问题，或者有任何建议，可以加入**唯一指定交流群：755612788**

## **最近更新**

- 增加了禁用单个命令的管理员命令，对所有群组生效
- 改善了初次使用的体验（增加扫码登陆，自动生成device.json和application.yaml）
- 增加了config配置下播推送，直播间标题推送，开启@全体成员，b站动态推送过滤器。
- 增加了config命令，目前支持配置推送同时@成员（实验阶段，并不保证稳定，目前只支持直播推送）。
- 增加了私聊的grant命令，现在可以通过私聊配置qq群内的群友命令权限了。
- 更新了list命令，现在list命令会展示所有网站的所有订阅，不必再指定-s和-t了。
- 增加了私聊的list命令，现在可以通过私聊查看订阅列表了。
- 增加了虎牙直播的推送。
- 增加了私聊的enable和disable命令，现在可以通过私聊配置qq群内的命令了。
- 增加了私聊的watch和unwatch命令，现在可以通过私聊配置qq群内的订阅了（注意仍然是qq群推送）。
- 更换船新b站监控方案，预计单帐号可支持1000订阅，最大延迟30秒。

## 已知问题

- 一些情况下无法正常识别QQ群管理员，属于MiraiGo问题，无法在本项目解决。
- b站动态被审核超过一段时间之后，该条动态可能不会推送。
- 订阅的帐号如果改名了，可能不会及时更新。

## 注意事项

- **bot只在群聊内工作，但命令可以私聊使用，避免在群内刷屏**（少数次要娱乐命令暂不支持，详细列表请看用法指南）
- **建议bot秘密码设置足够强，同时不建议把bot设置为QQ群管理员，因为存在密码被恶意爆破的可能（包括但不限于盗号、广告等）**
- **您应当知道，bot账号可以人工登陆，请注意个人隐私**
- bot掉线无法重连时将自动退出，请自行实现保活机制
- bot使用 [buntdb](https://github.com/tidwall/buntdb) 作为embed database，会在当前目录生成文件`.lsp.db`
  ，删除该文件将导致bot恢复出厂设置，可以使用 [buntdb-cli](https://github.com/Sora233/buntdb-cli) 作为运维工具，但注意不要在bot运行的时候使用（buntdb不支持多写）

## 敬告

- 请勿滥用
- 禁止商用

## 贡献

*Feel free to make your first pull request.*

想要为开源做一点微小的贡献？

[Golang点我入门！](https://github.com/justjavac/free-programming-books-zh_CN#go)

您也可以选择点一下右上角的⭐星⭐

发现问题或功能建议请到 [issues](https://github.com/Sora233/DDBOT/issues)

其他用法问题请到 [discussions](https://github.com/Sora233/DDBOT/discussions)

## 鸣谢

> [Goland](https://www.jetbrains.com/go/) 是一个非常适合Gopher的智能IDE，它极大地提高了开发人员的效率。

特别感谢 [JetBrains](https://jb.gg/OpenSource) 为本项目提供免费的 [Goland](https://www.jetbrains.com/go/) 等一系列IDE的授权

[<img src="https://user-images.githubusercontent.com/11474360/112592917-baa00600-8e41-11eb-9da4-ecb53bb3c2fa.png" width="200"/>](https://jb.gg/OpenSource)