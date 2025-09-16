# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [1.1.1] - 2025-09-13

### Changed

- add LICENSE.

Signed-off-by: 王布衣 <wangfengxy@sina.cn>

- 去掉windows.Handle的强制转换
- 补充IOCP缺失的GetQueuedCompletionStatusEx
- 补充连接ConnectEx的超时和连接成功的判断
- 实验性代码, 不能运行
- 完成一次iocp完整的网络操作
- 调整绑定socket到iocp句柄的completionKey为socket
- 新增一个http压力测试工具
- 更新go最低支持版本到1.25
- sort imports

## [1.1.0] - 2025-03-14

### Changed

- 新增实验性质的windows-iocp代码
- update changelog

## [1.0.22] - 2023-01-14

### Changed

- 更新gox版本号

## [1.0.21] - 2023-01-14

### Changed

- 修订部分package
- 优化部分代码
- 调整EventHandler和Event循环引用的告警
- 修复一处sliace的bug

## [1.0.20] - 2019-06-29

### Changed

- fix mod
- fix gox version

## [1.0.19] - 2019-03-30

### Changed

- add gitignore
- add README
- add source

[Unreleased]: https://gitee.com/quant1x/asio.git/compare/v1.1.1...HEAD

[1.1.1]: https://gitee.com/quant1x/asio.git/compare/v1.1.0...v1.1.1

[1.1.0]: https://gitee.com/quant1x/asio.git/compare/v1.0.22...v1.1.0

[1.0.22]: https://gitee.com/quant1x/asio.git/compare/v1.0.21...v1.0.22

[1.0.21]: https://gitee.com/quant1x/asio.git/compare/v1.0.20...v1.0.21

[1.0.20]: https://gitee.com/quant1x/asio.git/compare/v1.0.19...v1.0.20

[1.0.19]: https://gitee.com/quant1x/asio.git/releases/tag/v1.0.19
