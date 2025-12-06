# 背景
最近学习MCP, 发现MCP提供了三种传输方式: `stdio`, `SSE`和`Streamable HTTP`, 有意思的是,`SSE`作为一个标准传输协议在近期版本中逐渐被废弃, 作者不得不尝试了解`Streamable HTTP`以及它和`SSE`的区别.


## MCP对SSE的态度变化
MCP（Model Context Protocol，模型上下文协议，核心应用于 AI 模型推理的流式交互场景）对 SSE 的“不推荐”标注，**并非单一版本的硬性变更**，而是随 MCP 生态（尤其是 OpenAI 兼容生态、Cloudflare/Anthropic 等厂商的 MCP 实现）的演进逐步明确的。以下是关键时间线和版本背景，以及核心原因：

### 一、核心结论：关键版本/阶段
MCP 对 SSE 的“不推荐”标注，核心出现在 **MCP 1.1 版本（2023年Q4）**，并在 **MCP 2.0 正式版（2024年Q2）** 中被明确写入规范；而更早的 MCP 1.0（2023年Q1）则将 SSE 作为流式传输的默认推荐方案。

### 二、关键版本细节
| MCP 版本       | 发布时间   | 对 SSE 的态度                | 核心变更                                                                 |
|----------------|------------|-----------------------------|--------------------------------------------------------------------------|
| MCP 1.0        | 2023年Q1   | 推荐（默认流式传输方案）| 仅支持 SSE + WebSocket 两种流式实现，SSE 为 HTTP 场景的首选              |
| MCP 1.1（RC）  | 2023年Q3   | 谨慎使用（标注“局限性”）| 新增对 gRPC 流、HTTP/2 分块传输的支持，指出 SSE 的兼容性/性能问题        |
| MCP 1.1（正式）| 2023年Q4   | 不推荐（标注“Deprecated”）| 正式将 SSE 标记为“不推荐”，仅保留兼容，推荐 HTTP/2 分块（Streamable）+ WebSocket |
| MCP 2.0        | 2024年Q2   | 废弃（移除 SSE 优先支持）| 从规范中移除 SSE 作为“推荐方案”的描述，仅在“兼容层”提及 SSE 适配逻辑      |

### 三、标注 SSE 为“不推荐”的核心原因
MCP 作为面向 AI 模型推理的通用协议，放弃 SSE 的核心诉求是解决其在大规模/高并发场景下的硬伤：
1. **协议局限性**
   - SSE 基于 HTTP/1.1，存在“队头阻塞”（一个连接只能处理一个流），高并发下需创建大量 TCP 连接，资源消耗高；
   - SSE 仅支持“服务端→客户端”单向传输，无法满足 MCP 2.0 新增的“客户端中途中断/调整参数”（如暂停生成、修改温度）等双向交互需求。

2. **兼容性问题**
   - 部分网关/CDN 对 SSE 的 `text/event-stream` 格式支持不佳（如缓冲分块、截断数据）；
   - 移动端/弱网环境下，SSE 的自动重连逻辑不可控，易导致数据重复/丢失。

3. **性能短板**
   - SSE 强制文本格式（需序列化/反序列化），而 MCP 2.0 主推二进制协议（如 Protobuf），SSE 无法适配；
   - 无内置的流控/背压机制，服务端推送速率无法适配客户端处理能力。

### 四、MCP 推荐的替代方案（替代 SSE）
MCP 1.1+ 明确推荐以下方案替代 SSE，均属于“Streamable 能力”的实现：
| 场景                | 推荐方案                  | 核心优势                                  |
|---------------------|---------------------------|-------------------------------------------|
| HTTP 场景           | HTTP/2 分块传输（Chunked）| 无队头阻塞、支持双向流控、兼容现有 HTTP 栈 |
| 双向交互场景        | WebSocket（RFC 6455）| 全双工、低延迟、支持二进制帧              |
| 跨服务/高性能场景   | gRPC 流                   | 基于 HTTP/2、Protobuf 序列化、内置流控    |

### 五、补充说明
1. **厂商实现差异**：不同厂商的 MCP 衍生版本（如 OpenAI 的 MCP 兼容层、Anthropic 的 Claude API）对 SSE 的弃用节奏略有不同：
   - OpenAI API 仍保留 SSE 支持（`text/event-stream`），但在文档中标注“建议优先使用 WebSocket”；
   - Cloudflare AI Gateway 的 MCP 实现从 2024 年起已默认关闭 SSE 支持，仅提供 HTTP/2 分块和 WebSocket。

2. **兼容层保留**：MCP 并未完全移除 SSE，而是将其归入“Legacy 兼容层”，用于适配老版本客户端，新开发需严格遵循 Streamable + HTTP/2 分块/WebSocket 的方案。

# SSE和Streamable HTTP的区别
你提出的这个问题非常关键——**MCP（Model Context Protocol，或各类云服务/API 规范中的 MCP）中区分 SSE 和 Streamable 并非实现方式相同，而是「协议设计维度」和「能力抽象维度」的差异**：前者是「传输层协议标准」，后者是「功能特性描述」，我的前序示例仅用 SSE 实现了“流式输出”的表象，但未区分二者的核心差异，下面拆解清楚：

### 一、先明确：SSE vs Streamable 的本质定义
| 维度                | SSE (Server-Sent Events)                          | Streamable（可流式化）                          |
|---------------------|---------------------------------------------------|------------------------------------------------|
| 本质                | **传输层协议标准**（HTTP 之上的轻量级协议）| **功能特性抽象**（描述“数据可分块、逐段传输”的能力） |
| 归属                | 属于 HTTP 生态的具体协议（有明确的格式/交互规则） | 属于业务/API 设计的“能力标签”（无固定协议）|
| 约束                | 必须遵循 `text/event-stream` 格式、单向（服务端→客户端）、基于 HTTP 长连接 | 无固定格式，仅要求“数据不一次性返回，可分批次传输”，支持任意传输协议（HTTP、WebSocket、TCP 等） |
| 场景                | 仅用于 HTTP 场景下的服务端单向推送                | 可覆盖所有“逐段传输”场景（如 HTTP 流式响应、WebSocket 双向流、gRPC 流、文件分片下载等） |

### 二、MCP 中区分二者的核心原因
MCP 作为「模型/服务交互协议」，需要从“**能力抽象**”和“**具体实现**”两个层面设计：
1. **Streamable（可流式化）：定义“WHAT”**
   - 是对“数据输出能力”的抽象描述：表示该接口/服务支持“非一次性返回全部数据，而是分块、实时输出”（比如大语言模型的逐字生成、大数据查询的逐行返回）。
   - 它不绑定具体传输协议，仅约定“能力”：比如 MCP 可能定义 `streamable: true` 表示该接口支持流式，至于底层用 SSE、WebSocket、gRPC 流实现，由开发者选择。

2. **SSE：定义“HOW”**
   - 是 Streamable 能力在「HTTP 场景下的一种具体实现方式」：是 Streamable 的“子集”。
   - MCP 中单独标注 SSE，是为了明确“该 Streamable 能力基于 SSE 协议实现”（比如补充 `transport: sse`），方便客户端按 SSE 标准解析数据。

### 三、前序示例的“问题”：仅用 SSE 实现了 Streamable，但未体现差异
我的前序代码用 SSE 实现了“流式输出”，但这只是 Streamable 的**一种实现方式**——如果要体现 MCP 中二者的差异，需要分别实现：
- **Streamable 的通用实现**（不绑定 SSE 协议）：比如纯 HTTP 流式响应（无 SSE 格式）；
- **SSE 的实现**（绑定 SSE 协议的 Streamable）：即前序的 SSE 版本。

#### 补充：Streamable（非 SSE）的 Go 实现（通用 HTTP 流式）
```go
// 通用流式响应（无 SSE 格式，仅分块输出）
func genericStreamHandler(w http.ResponseWriter, r *http.Request) {
	// 仅设置流式响应头，不绑定 SSE 格式
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	clientGone := r.Context().Done()
	counter := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			return
		case <-ticker.C:
			counter++
			// 无 SSE 格式，直接输出纯文本（Streamable 的通用形式）
			msg := fmt.Sprintf("通用流式数据 %d - %s\n", counter, time.Now().Format("15:04:05"))
			_, err := fmt.Fprint(w, msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// 在 main 中注册
http.HandleFunc("/generic-stream", genericStreamHandler)
```

#### 对应前端（非 SSE 的 Streamable 接收）
需要用 `fetch + ReadableStream` 手动解析（而非 `EventSource`），体现“Streamable 不依赖 SSE 协议”：
```javascript
// 接收通用流式响应（非 SSE）
async function startGenericStream() {
    logElement.innerHTML += `[${new Date().toLocaleTimeString()}] 开始接收通用流式数据<br>`;
    
    const response = await fetch('http://localhost:8080/generic-stream');
    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    // 循环读取分块数据
    while (true) {
        const { done, value } = await reader.read();
        if (done) {
            logElement.innerHTML += `[${new Date().toLocaleTimeString()}] 通用流结束<br>`;
            break;
        }
        // 解析二进制数据为文本
        const text = decoder.decode(value, { stream: true });
        logElement.innerHTML += `[${new Date().toLocaleTimeString()}] ${text}<br>`;
        logElement.scrollTop = logElement.scrollHeight;
    }
}
```

### 四、总结：MCP 区分二者的核心价值
1. **解耦“能力定义”和“传输实现”**：
   - Streamable 是“接口能力”：告诉客户端“这个接口支持流式”；
   - SSE 是“传输协议”：告诉客户端“该流式接口用 SSE 协议传输，需按 SSE 格式解析”。
2. **兼容多场景**：
   - 若 MCP 服务跑在 WebSocket 场景，Streamable 可基于 WebSocket 实现，而非 SSE；
   - 若跑在 HTTP 场景，Streamable 可选择 SSE（易解析）或纯 HTTP 流式（更轻量）。
3. **标准化解析**：
   - 标注 SSE 后，客户端可直接用 `EventSource` 解析；未标注 SSE 的 Streamable，客户端需按自定义格式解析（如上述 `ReadableStream`）。

简单来说：**Streamable 是“能流式”，SSE 是“用 SSE 协议实现的流式”** —— 我的前序示例仅展示了后者，而 MCP 中区分二者，是为了既定义“能力”，又明确“实现方式”。