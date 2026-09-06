// Package auth 是一个可移植、与登录供应商无关的认证模块。
//
// 分层方式与 modules/billing、modules/email 一致：
//
//	domain/   纯类型：Identity、Token、OAuthProfile、错误
//	event/    领域事件：UserSignedUp、UserLoggedIn
//	port/     接口：登录供应商、验证码、令牌、用户仓储、角色解析、事件总线
//	adapter/  具体实现
//	  jwt/         HS256 令牌和 WebSocket ticket
//	  google/      Google OAuth
//	  github/      GitHub OAuth
//	  emailcode/   无密码邮箱验证码登录，发送能力来自 modules/email
//	  memstore/    内存验证码限流、OAuth flow 和交换码，仅用于开发与测试
//	  gormstore/   GORM 验证码、每日次数、OAuth flow 和交换码
//	  eventbus/    进程内同步事件总线
//	app/      用例：发送/校验验证码、OAuth、令牌交换与刷新、WebSocket ticket
//	http/     Gin 处理器、认证中间件和路由挂载
//	auth.go   模块组装
//
// 宿主项目负责提供：
//  1. port.UserStore         — 读写宿主自己的用户表
//  2. port.RoleResolver      — 决定登录身份的角色
//  3. port.OAuthFlowStore     — 原子保存/消费浏览器绑定 OAuth state
//  4. port.ExchangeCodeStore  — 保存浏览器绑定 OAuth 回调交换码
//  5. 事件监听器               — 响应 UserSignedUp / UserLoggedIn
//
// 认证模块不会导入任何项目专属模型。
package auth

// Module、Deps 和 New() 位于 module.go；本文件只说明包结构。
