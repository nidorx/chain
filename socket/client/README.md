# Chain Socket Client

TypeScript client library for Chain's real-time socket channels.

## Overview

The Chain client connects to Chain socket servers over **Server-Sent Events (SSE)**. It supports:

- **Multiplexed channels** — multiple topics over a single connection
- **Cluster-aware connectivity** — automatic node discovery and failover via `getNodes`
- **Automatic reconnection** — exponential backoff with configurable intervals
- **Message deduplication** — pluggable deduplication via the `duplicated` option and `History` class
- **Transport abstraction** — extensible transport layer (SSE implemented, LongPolling reserved)

## Installation

The client is served as a static endpoint by the Chain server. Include it directly in your HTML:

```html
<script src="/chain.js" type="module"></script>
```

Or use the TypeScript source with a build step:

```typescript
import * as chain from '/chain.js';
```

### Server Setup

On the Go side, register the client endpoints with `socket.ClientJsHandler`:

```go
socket.ClientJsHandler(router, "/socket")
```

This registers three endpoints:

| Endpoint       | Content-Type                        | Description                          |
|----------------|-------------------------------------|--------------------------------------|
| `/chain.ts`    | `application/typescript`            | Raw TypeScript source                |
| `/chain.js`    | `text/javascript`                   | Compiled JavaScript                  |
| `/chain.js.map`| `application/json`                  | Source map                           |

All responses include `ETag` headers for HTTP caching.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                        Browser                           │
│                                                          │
│  ┌────────────┐    ┌───────────────┐    ┌─────────────┐  │
│  │  Socket    │───>│  Channel(s)   │───>│  Push       │  │
│  │ (connects, │    │  (join,       │    │  (send msg, │  │
│  │  discovers │    │   push,       │    │   timeout,  │  │
│  │  nodes)    │    │   on, leave)  │    │   retry)    │  │
│  └─────┬──────┘    └───────┬───────┘    └──────┬──────┘  │
│        │                   │                   │         │
│        ▼                   ▼                   ▼         │
│  ┌────────────┐    ┌───────────────┐    ┌─────────────┐  │
│  │ Transport  │    │  Events       │    │  Retry      │  │
│  │ (SSE)      │    │  (on, off,    │    │  (backoff,  │  │
│  │            │    │   emit)       │    │   intervals)│  │
│  └────────────┘    └───────────────┘    └─────────────┘  │
└─────────┬────────────────────────────────────────────────┘
          │ SSE (EventSource) + fetch POST
          ▼
┌──────────────────────────────────────────────────────────┐
│                    Chain Server                          │
│  ┌──────────┐  ┌─────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Handler  │─>│ Session │─>│  Socket  │─>│ Channel  │  │
│  │ (routes) │  │ (state) │  │ (joined) │  │ (pubsub) │  │
│  └──────────┘  └─────────┘  └──────────┘  └──────────┘  │
└──────────────────────────────────────────────────────────┘
```

## Classes

### Socket

The `Socket` class manages the connection to the server cluster and channels.

#### Constructor

```typescript
const socket = new chain.Socket(options: SocketOptions);
```

#### SocketOptions

| Option                  | Type                      | Default                          | Description                                                                 |
|-------------------------|---------------------------|----------------------------------|-----------------------------------------------------------------------------|
| `timeout`               | `number`                  | `30000`                          | Default timeout in ms for push operations                                   |
| `sessionStorage`        | `Storage`                 | `window.sessionStorage`          | Browser storage for socket ID persistence across page navigations           |
| `rejoinInterval`        | `number[]`                | `[1000, 2000, 5000, 10000]`      | Retry intervals (ms) for rejoining channels after errors                    |
| `disconnectIdleTimeout` | `number`                  | `5000`                           | Disconnect from server after this delay (ms) when no channels remain        |
| `transport`             | `TransportConfig[]`       | `[{name: "SSE"}]`               | Transport configurations to use for server connection                       |
| `getNodes`              | `() => Promise<string[]>` | **required**                     | Async function that returns the list of server endpoint URLs               |
| `getNodesInterval`      | `number`                  | `30`                             | How often (seconds) to poll `getNodes` for server list changes              |
| `dropNodeConnectionAfter`| `number`                 | `5000`                           | Delay (ms) before dropping the old node connection after switching nodes    |
| `duplicated`            | `(msg: Message) => boolean`| `() => false`                   | Function to detect and discard duplicate messages                           |

#### TransportConfig

| Option   | Type      | Default     | Description                                              |
|----------|-----------|-------------|----------------------------------------------------------|
| `name`   | `string`  | **required**| Transport name (e.g., `"SSE"`)                           |
| `sid`    | `string`  | auto-generated | Session ID (assigned automatically per browser tab)   |
| `cors`   | `boolean` | `false`     | Enable CORS mode for cross-origin requests               |
| `params` | `any`     | `{}`        | Additional query parameters for the transport connection  |

#### Methods

| Method                                  | Description                                              |
|-----------------------------------------|----------------------------------------------------------|
| `connect()`                             | Initiate the socket connection                           |
| `disconnect()`                          | Close the socket connection                              |
| `isConnected(): boolean`                | Returns `true` if the socket is connected                |
| `channel(topic, params?, options?)`     | Create and return a new Channel for the given topic      |
| `leave(topic)`                          | Leave a channel by topic name                            |
| `getTimeout(): number`                  | Get the configured push timeout                          |
| `getRejoinInterval(): number[]`         | Get the configured rejoin intervals                      |

#### Events

| Event       | Payload          | Description                                      |
|-------------|------------------|--------------------------------------------------|
| `open`      | `Transport`      | Emitted when the transport connects successfully |
| `close`     | —                | Emitted when the transport disconnects           |
| `error`     | `any`            | Emitted on transport error                       |
| `message`   | `Message`        | Emitted for every received message               |
| `message:duplicated` | `Message` | Emitted when a message is identified as duplicate |

#### Cluster Node Discovery

The `getNodes` callback enables cluster-aware connectivity. The client periodically calls this function to discover available server nodes and automatically migrates connections to the highest-priority node:

```typescript
const socket = new chain.Socket({
    getNodes: async () => {
        const response = await fetch('/api/nodes');
        const nodes = await response.json();
        // Returns full endpoint URLs, e.g.:
        // ["http://server1:8080/socket", "http://server2:8080/socket"]
        return nodes;
    },
    getNodesInterval: 30, // check every 30 seconds
});
```

When the node list changes, the client:
1. Connects to the new preferred node
2. Waits `dropNodeConnectionAfter` ms before closing the old connection
3. This delay handles message deduplication during the transition

#### Session ID Management

Each browser tab gets a unique socket ID (`sid`), persisted in `sessionStorage` so it survives page navigations within the same tab. The `sid` is sent as a query parameter on every transport connection, enabling session recovery on reconnect.

---

### Channel

A `Channel` represents a bidirectional communication path on a specific topic. Channels must be joined before sending or receiving messages.

#### Constructor

Created via `socket.channel(topic, params?, options?)`:

| Parameter | Type           | Description                                    |
|-----------|----------------|------------------------------------------------|
| `topic`   | `string`       | Topic name (commas and `*` not allowed)        |
| `params`  | `any`          | Join parameters sent to the server             |
| `options` | `ChannelOptions`| Optional hooks (`onMessage`)                  |

#### ChannelOptions

| Option      | Type                                                        | Description                                                        |
|-------------|-------------------------------------------------------------|--------------------------------------------------------------------|
| `onMessage` | `(event: string, payload: any, ref?: number, joinRef?: number) => any` | Hook invoked for every incoming message. Must return the payload (modified or unmodified). Return `null`/`undefined` throws an error. |

#### Methods

| Method                                      | Returns  | Description                                                |
|---------------------------------------------|----------|------------------------------------------------------------|
| `join(timeout?)`                            | Channel  | Join the channel. Returns `this` for chaining. Can only be called once per instance. |
| `push(event, payload, timeout?)`            | `Push`   | Send a message to the channel. Returns a `Push` for reply handling. |
| `leave(timeout?)`                           | `Push`   | Leave the channel. Returns a `Push` for leave acknowledgement. |
| `on(event, callback)`                       | Channel  | Subscribe to an event on this channel                      |
| `onClose(callback)`                         | Channel  | Subscribe to the channel close event                       |
| `onError(callback)`                         | Channel  | Subscribe to the channel error event                       |
| `getTopic(): string`                        | `string` | Get the channel topic name                                 |
| `getJoinRef()`                              | `number` | Get the join reference (unique per join attempt)           |
| `canPush(): boolean`                        | `boolean`| Returns `true` if the socket is connected and channel joined |
| `isClosed()` / `isErrored()` / `isJoined()` / `isJoining()` / `isLeaving()` | `boolean` | State checkers |

#### Events

| Event            | Payload     | Description                                          |
|------------------|-------------|------------------------------------------------------|
| `ok`             | `any`       | Channel successfully joined                          |
| `error`          | `any`       | Channel join failed                                  |
| `timeout`        | —           | Channel join timed out                               |
| `message`        | `any`       | Any message received on this channel                 |
| `chan_reply_N`   | `any`       | Reply to a `push` with ref `N` (status: `ok`/`error`) |
| `_close`         | `string`    | Internal: channel closed                             |
| `_error`         | `any`       | Internal: channel error with reason                  |

#### Lifecycle

```
  channel(topic) ──> [CLOSED]
                        │
                   join() │
                        ▼
                   [JOINING] ───────────────────────────┐
                        │                               │
              ┌─────────┼─────────┐                     │
              ▼         ▼         ▼                     │
           [JOINED] [ERRORED] [TIMEOUT]                 │
              │         │         │                     │
              │    rejoin()  resend()                   │
              │         │         │                     │
              │         └────┬────┘                     │
              │              ▼                           │
              │         [JOINING] ◄─────────────────────┘
              │
         leave() │
              ▼
          [LEAVING]
              │
              ▼
          [CLOSED]
```

---

### Push

A `Push` represents a single outgoing message with reply handling.

#### Methods

| Method                            | Returns | Description                                            |
|-----------------------------------|---------|--------------------------------------------------------|
| `on(event, callback)`             | `Push`  | Register a reply handler (`ok`, `error`, `timeout`)    |
| `send()`                          | void    | Send the message (starts timeout timer)                |
| `resend(timeout)`                 | void    | Re-send with a new timeout                             |
| `getRef(): number`                | `number`| Get the message reference number                       |
| `getTimeout(): number`            | `number`| Get the configured timeout                             |

#### Reply Handling

```typescript
channel.push("event", { data: "value" })
    .on("ok",      (payload) => console.log("server replied:", payload))
    .on("error",   (err)     => console.log("server errored:", err))
    .on("timeout", ()        => console.log("push timed out"));
```

---

### TransportSSE

The default transport using Server-Sent Events.

- **Outbound messages**: sent via `fetch POST` (fire-and-forget)
- **Inbound messages**: received via `EventSource` stream
- **Session recovery**: uses `sid` parameter for session resumption

Two endpoints are derived from the base endpoint URL:

| Endpoint    | Method | Purpose                     |
|-------------|--------|-----------------------------|
| `{endpoint}/sse`  | GET    | EventSource stream (inbound) |
| `{endpoint}/sse`  | POST   | Push messages (outbound)     |

Both include `sid` as a query parameter for session identification.

---

### Retry

Exponential backoff retry utility used internally by `Socket` and `Channel` for reconnection.

```typescript
const retry = new Retry(
    (tries) => { /* callback invoked on each retry */ },
    [1, 500, 1000, 2000, 5000] // intervals in ms
);

retry.retry();  // schedule next retry
retry.reset();  // cancel pending retry, reset counter
retry.tries();  // number of retries so far
```

---

### History

Sliding-window deduplication helper. Keeps message keys in memory for a configurable TTL.

```typescript
const history = new chain.History(5); // keep 5 seconds of history

const socket = new chain.Socket({
    duplicated: (msg: Message) => history.exists(msg.payload.messageId)
});
```

The `History` class rotates its internal storage every second, discarding entries older than the configured TTL.

---

### Events (Base Class)

Lightweight event emitter used as the base for `Socket`, `Channel`, and `Transport`.

| Method                         | Description                        |
|--------------------------------|------------------------------------|
| `on(event, callback): T`       | Subscribe to an event              |
| `off(event, callback)`         | Unsubscribe from an event          |
| `emit(event, ...args)`         | Fire an event to all subscribers   |

---

### Message

The wire format for all messages exchanged between client and server.

```typescript
interface Message {
    joinRef: number;    // Join reference (identifies the channel join instance)
    ref: number;        // Message reference (identifies individual pushes)
    topic?: string;     // Channel topic
    event: string;      // Event name
    kind?: MessageKindEnum;  // PUSH(0), REPLY(1), BROADCAST(2)
    payload: any;            // Decoded payload
    payload_raw: string;     // Raw JSON payload string
}
```

#### Wire Format (JSON array)

All messages are encoded as JSON arrays:

```
[kind, joinRef, ref, topic, event, payload]
```

- **PUSH** (kind=0): `[0, joinRef, ref, topic, event, payload]`
- **REPLY** (kind=1): `[1, joinRef, ref, status, response_payload]` — decoded as `{status: 'ok'|'error', response}`
- **BROADCAST** (kind=2): `[2, topic, event, payload]` — topic and event come from different positions

---

### Global Options

```typescript
chain.Options.Debug = true;         // Enable all debug logging (default: true)
chain.Options.DebugSocket = true;   // Socket-specific debug
chain.Options.DebugChannel = true;  // Channel-specific debug
chain.Options.DebugTransport = true;// Transport-specific debug
```

Set to `false` to suppress console output.

---

## Usage Examples

### Basic Chat Client

```javascript
import * as chain from '/chain.js';

const socket = new chain.Socket({
    getNodes: async () => ['/socket']
});
socket.connect();

const channel = socket.channel("chat:lobby", { user: "alice" });

channel.join()
    .on('ok', () => chain.log('Join', "success"))
    .on('error', err => chain.log('Join', "errored", err))
    .on('timeout', () => chain.log('Join', "timed out"));

// Send a message
channel.push('shout', { name: 'Alice', body: 'Hello!' });

// Receive messages
channel.on('shout', (message) => {
    console.log(`${message.name}: ${message.body}`);
});
```

### Cluster-Aware Client

```javascript
const socket = new chain.Socket({
    transport: [{ name: "SSE", cors: true }],
    getNodesInterval: 5,
    getNodes: async () => {
        const res = await fetch('/api/nodes');
        const ports = await res.json();
        const host = window.location.hostname;
        return ports.map(port => `http://${host}:${port}/socket`);
    }
});
socket.connect();

// Join a channel — the client automatically routes to the correct node
socket.channel("chat:lobby")
    .on('message', handleMessage)
    .join();
```

### With Message Deduplication

```javascript
const history = new chain.History(5); // 5-second window

const socket = new chain.Socket({
    getNodes: async () => ['/socket'],
    duplicated: (msg) => {
        // Assume payload has a unique messageId field
        return history.exists(msg.payload.messageId);
    }
});
```

### Leaving a Channel

```javascript
const push = channel.leave();
push.on('ok', () => console.log('Left successfully'));
push.on('timeout', () => console.log('Leave timed out'));
```

### Custom onMessage Hook

```javascript
const channel = socket.channel("room:42", {}, {
    onMessage: (event, payload, ref, joinRef) => {
        // Intercept all incoming messages
        if (event === 'new_msg') {
            payload.timestamp = Date.now();
        }
        return payload; // must return the payload
    }
});
```

## Error Handling

- **Join failures**: listen to `channel.join().on('error', ...)` and `channel.join().on('timeout', ...)`
- **Push failures**: listen to `push.on('error', ...)` and `push.on('timeout', ...)`
- **Socket errors**: listen to `socket.on('error', ...)` — triggers channel rejoin retries
- **Network loss**: the `Retry` class handles exponential backoff for both socket reconnection and channel rejoin

## See Also

- [Server-side Socket documentation](../docs/SOCKET.md)
- [Architecture overview](../docs/architecture.md)
- [Socket examples](../examples/socket-chat/)
- [Cluster example](../examples/socket-cluster/)
