# DeepSeek Harness 反向代理与网关适配技术文档

本文档记录在飞牛 NAS 系统中适配 **DeepSeek Harness (DSH)** 反向代理与网关子路径时的技术方案与实现细节，供维护与版本升级参考。

---

## 一、设计原则

1. **零侵入**：不修改 DSH 官方 NPM 运行时（`@deepseek-ai/dsh`），保持上游代码纯净，便于后续直接升级；
2. **多模式支持**：
   - **飞牛网关模式**：子路径代理（`http://<NAS_IP>:5666/app/deepseek-harness/fngateway/`）；
   - **独立代理模式**：独立端口（`http://<NAS_IP>:2299/`）；
   - **本地回环模式**：本地调试（`http://127.0.0.1:2298/`）；
3. **内聚收敛**：所有适配逻辑由 Go 代理服务层（[`proxy.go`](./proxy.go)、[`fngateway.go`](./fngateway.go)、[`harness.go`](./harness.go)）处理。

---

## 二、问题排查与解决对照表

| 序号 | 现象 / 问题 | 根本原因 | 解决方案 | 涉及文件 |
| :--- | :--- | :--- | :--- | :--- |
| **1** | 特权 API（配置读写、模型发现等）返回 **403 Forbidden** | 后端通过 `isTrustedApiRequest` 校验请求头，仅允许来自本地回环且同源的请求 | 在反向代理 `Rewrite` 阶段将请求头改写为目标同源 `Host`、`Origin`，并将 `Sec-Fetch-Site` 设为 `same-origin` | `proxy.go`<br>`fngateway.go` |
| **2** | 飞牛网关子路径下静态资源与接口 **404** | 前端基于根路径 `/` 构建，子路径环境下静态标签、动态请求及 WebSocket 未携带网关前缀 | 1. 代理响应 HTML 时正则替换静态属性（`src`/`href`）；<br>2. 注入网关桥接脚本拦截 `fetch`、`XHR`、`WebSocket`（含 `/api/remote.mux`）、DOM 插入等动态请求 | `fngateway.go` |
| **3** | HTTP 局域网访问报错 `randomUUID is not a function` | 浏览器限制 `crypto.randomUUID()` 仅在安全上下文（HTTPS/localhost）可用，普通 HTTP 局域网 IP 缺失该 API | 在 HTML 头部注入基于 RFC4122 v4 的纯 JS UUID 生成器作为兜底 polyfill | `proxy.go`<br>`fngateway.go` |
| **4** | 远程访问下「插件配置」面板空白、模型设置无法读取保存 | 前端 `@deepseek-ai/dsh-client-connection` 依据 `location.hostname` 判定 `isLoopback`；非回环 IP 时进入 memory 模式并拒绝向后端请求配置 | 页面注入 `window.__DSH_TRANSPORT__ = { ownsHost: true }`，走通官方原生特权分支（使 `isLoopback` 判定为 `true`） | `proxy.go`<br>`fngateway.go` |
| **5** | 页面右上角显示红字“无法打开配置文件” | 该按钮尝试调用宿主机图形界面编辑器（如 Linux 下的 `xdg-open`），在 NAS 无头（Headless）环境下必然失败并报错 | 注入 CSS 样式 `<style>[data-slot="settings.action"] { display: none !important; }</style>` 隐藏该按钮 | `proxy.go`<br>`fngateway.go` |
| **6** | 会话头部出现“在 Zed 中打开”等桌面应用分体按钮 | 上游新增 `open-in-app` 功能；后台无 SSH 标记时，后端探测本地已安装应用并向前端下发列表 | 全局环境初始化 `InitAppEnv()` 注入 `SSH_CONNECTION=127.0.0.1 0 127.0.0.1 22`，触发官方远程环境判定，自动隐藏该桌面按钮 | `config.go` |
| **7** | 移动端 App 提示鉴权失效或报错 `HTML did not preload client.js` | 1. 官方 Cookie 为 `SameSite=Strict`，部分移动端 WebView 无法携带凭据导致 401；<br>2. HTML 无防缓存头，导致 WebView 强缓存旧 `rev` 版本资源；<br>3. 移动端网络栈可能折叠 `/plugins/??` 连续问号引发 404 | 1. **服务端代持会话**：反代层捕获 Token 向回环换票并由 Go 内存代持，转发自动注入官方 Cookie；<br>2. **禁用强缓存**：HTML 响应强制写入 `Cache-Control: no-store`；<br>3. **问号还原**：转发前自动补齐被折叠的首个问号 | `harness.go`<br>`proxy.go`<br>`fngateway.go` |
| **8** | 飞牛网关子路径下上传附件失败（报 404） | 客户端 `@deepseek-ai/dsh-client-file-upload` 使用 `new URL('/api/...', location.origin)` 拼接地址导致脱落网关子路径前缀；且默认在独立 Web Worker 中发送，绕过了主线程网络拦截 | 注入上游官方原生预留的 `window.__DSH_FILE_UPLOAD__ = { fetch: ... }` 契约，补齐网关前缀并统一由主线程代理管道发送 | `fngateway.go` |

---

## 三、核心技术实现细节

### 1. 反向代理标头伪装与上下文配置
在 [`proxy.go`](./proxy.go) 中改写目标标头以通过 DSH 本地特权校验，并关闭代理层压缩以支持响应体改写：
```go
pr.Out.Header.Set("Host", targetURL.Host)
pr.Out.Header.Set("Origin", "http://"+targetURL.Host)
pr.Out.Header.Set("Sec-Fetch-Site", "same-origin")
pr.Out.Header.Set("Accept-Encoding", "identity")
```

### 2. 客户端原生特权契约声明（ownsHost）
DSH 上游 `@deepseek-ai/dsh-client-connection` 提供了针对嵌入式宿主的声明字段：
```typescript
// 上游计算逻辑
isLoopback: transport?.ownsHost === true || pageLocation === undefined || isLoopbackHostname(pageLocation.hostname)
```
通过在 HTML 头部注入以下代码，直接启用特权模式以读取和保存配置：
```javascript
try {
  window.__DSH_TRANSPORT__ = Object.assign(window.__DSH_TRANSPORT__ || {}, { ownsHost: true });
} catch (_) {}
```

### 3. 子路径网关路由桥接（fngateway.go）
在飞牛网关子路径反代环境下，通过前置脚本对前端网络与 DOM 环境进行路径适配：
- **基准路径**：注入 `<base href="...">` 规范相对路径解析；
- **路由与跳转**：拦截 `location.pathname` 读写及 `history.pushState` / `replaceState`，避免单页路由跳出网关前缀；
- **网络请求**：改写 `fetch`、`XMLHttpRequest.prototype.open`、`WebSocket`（含 `/api/remote.mux`）及 `EventSource` 请求路径；
- **DOM 资源**：拦截动态插入的 `<script>`、`<img>`、`<link>` 等标签路径，适配动态加载模块；
- **PWA 清单**：重写 `manifest.webmanifest` 中的 `scope` 与 `start_url`。

### 4. 附件上传网关子路径适配（__DSH_FILE_UPLOAD__）
官方 `@deepseek-ai/dsh-client-file-upload` 在页面载体扩展中预留了 `__DSH_FILE_UPLOAD__` 钩子：
```typescript
const hook = (globalThis as ClientFileUploadGlobal).__DSH_FILE_UPLOAD__
this.transport = hook === undefined ? workerTransport() : customTransport(hook.fetch)
```
通过前置注入该官方钩子，将原本在独立 Web Worker 内发送的请求收敛回页面主线程，自动通过 `toGatewayUrl` 补齐网关前缀并携带凭据：
```javascript
targetWindow.__DSH_FILE_UPLOAD__ = {
  fetch: function (inputUrl, init) {
    var mapped = toGatewayUrl(inputUrl);
    var finalInit = init || {};
    if (!finalInit.credentials) finalInit.credentials = "include";
    return targetWindow.fetch(mapped !== null ? mapped.toString() : inputUrl, finalInit);
  }
};
```

### 5. 无头桌面控件适配
- **隐藏打开配置文件按钮**：
  ```html
  <style>[data-slot="settings.action"] { display: none !important; }</style>
  ```
- **注入远程环境标记**：
  在 `config.go` 初始化中设置环境变量：
  ```go
  _ = os.Setenv("SSH_CONNECTION", "127.0.0.1 0 127.0.0.1 22")
  ```
  使 DSH 将运行上下文识别为远程环境，关闭桌面应用打开（`open-in-app`）及图形弹窗选择器。

### 6. 服务端会话代持与契约容错
针对移动端 WebView 的环境差异提供稳定性保障：
- **会话代持**：反向代理捕获客户端的登录 Token 后，直接在 Go 服务端内存中换取并持有 `dsh-auth-*` Cookie，转发至上游时统一注入，解除对客户端 Cookie 存储能力的依赖；
- **禁用 HTML 缓存**：响应头强制设置 `Cache-Control: no-store, no-cache, must-revalidate`，避免前端加载失效的旧版本资产；
- **URL 问号还原**：对请求路径形如 `/plugins/?/...` 的请求自动还原为 `/plugins/??/...`，保障上游模块打包器路由匹配。

---

## 四、排查与验证方法

### 1. 验证设置与特权 API 状态
在浏览器控制台执行以下请求，若返回配置信息且 `ok: true`，说明特权通道已正常打通：
```javascript
fetch('/api/settings.describe', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ type: 'client-request', rpcId: 'test', method: 'settings.describe', payload: {} })
}).then(r => r.json()).then(console.log);
```

### 2. 上游升级排查要点
- 更新 DSH 核心包时，无需重新构建前端代码；
- 若配置面板或接口异常，优先确认上游 `ownsHost` 声明字段及 RPC 路由是否有破坏性变更。
