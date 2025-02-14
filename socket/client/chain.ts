/**
 * Based on https://github.com/phoenixframework/phoenix/tree/main/assets/js/phoenix
 */

export const Options: {
    Debug: boolean; // When true, enables debug logging. Default false.
    [key: string]: any
} = {
    Debug: true,
}

export interface SocketOptions {
    // The default timeout in milliseconds to trigger push timeouts. (Defaults 30000)
    timeout?: number;

    sessionStorage?: Storage;

    rejoinInterval?: number[];

    disconnectIdleTimeout?: number;

    /**
     * The transport mechanisms for server connection
     * 
     * Default [{name:"SSE"}, {name:"LongPolling"}]
     */
    transport?: TransportConfig[];

    /**
     * Gets the list of socket servers. Allows load balancing or fine-tuned connection rules.
     * 
     * E.g., Retrieve a list of backend servers grouping users by region or business rule.
     */
    getNodes: () => Promise<string[]>;

    /**
     * How often to check for modifications in the server list.
     * 
     * Default is 30 seconds.
     */
    getNodesInterval?: number;

    /**
     * After connecting to a new node, disconnects from the old server (if still active) after this interval.
     * 
     * Important to handle message deduplication.
     * 
     * Default is 5 seconds.
     */
    dropNodeConnectionAfter?: number;

    /**
     * Allows discarding duplicate messages based on a specific rule.
     * 
     * @param msg 
     * @returns 
     */
    duplicated?: (msg: Message) => boolean;
}

export interface TransportConfig {
    name: string;
    sid?: string;
    cors?: boolean;
    params?: any;
}

export interface ChannelOptions {
    onMessage?: (_e: string, payload: any, ref?: number, p_joinRef?: number) => any;
}

export enum MessageKindEnum {
    PUSH = 0,
    REPLY = 1,
    BROADCAST = 2
}

enum ChannelStateEnum {
    CLOSED = 0,
    ERRORED = 1,
    JOINED = 2,
    JOINING = 3,
    LEAVING = 4,
}

enum SocketStateEnum {
    DISCONNECTED = 0,
    CONNECTING = 2,
    CONNECTED = 3,
    DISCONNECTING = 4,
    ERRORED = 5,
}

export interface Message {
    joinRef: number;
    ref: number;
    topic?: string;
    event: string;
    kind?: MessageKindEnum;
    payload: any | string | { status: 'ok' | 'error'; response: string };
    payload_raw: string;
}

const SOCKET = 'Socket';
const CHANNEL = 'Channel';
const TRANSPORT = 'Transport';

/**
 * Copyright 2016 Andrey Sitnik <andrey@sitnik.ru>, https://github.com/ai/nanoevents/blob/main/LICENSE
 */
export class Events<T> {
    private readonly events: { [key: string]: Array<(...args: any) => void> } = {};

    emit(event: string, ...args: any) {
        for (let callbacks = this.events[event] || [], i = 0, l = callbacks.length; i < l; i++) {
            callbacks[i](...args);
        }
    }

    on(event: string, callback: (...args: any) => void): T {
        this.events[event]?.push(callback) || (this.events[event] = [callback]);
        return (this as any);
    }

    off(event: string, callback: any) {
        let callbacks = this.events[event];
        if (callbacks) {
            let idx = callbacks.indexOf(callback);
            if (idx >= 0) {
                callbacks.splice(idx, 1);
            }
        }
    }
}

/**
 * Represents a node in a server cluster.
 */
interface Node {
    retry: Retry
    endpoint: string,
};

/**
 * Initializes the Socket
 */
export class Socket extends Events<Socket> {

    private ref = 1;
    private state: SocketStateEnum;
    private endpoint: string; // current node endpoint
    private transport: Transport; // current transport
    private transportListOld: Transport[];
    private disconnectIdleTimer: any;
    private readonly timeout: number;
    private readonly channels: Channel[] = [];
    private readonly duplicated: (msg: Message) => boolean;
    private readonly sendBuffer: Array<() => void> = [];
    private readonly sessionStorage: Storage;
    private readonly rejoinInterval: number[];
    private readonly transportConfig: TransportConfig[];
    private readonly disconnectIdleTimeout: number;
    private readonly dropNodeConnectionAfter: number;

    constructor(options: SocketOptions) {
        super();

        this.state = SocketStateEnum.DISCONNECTED;
        // this.endpoint = endpoint;
        this.timeout = options.timeout || 30000;
        this.sessionStorage = options.sessionStorage || (window.sessionStorage);
        this.transportConfig = options.transport || [{ name: "SSE" }]
        this.rejoinInterval = options.rejoinInterval || [1000, 2000, 5000, 10000];
        this.transportListOld = [];
        this.disconnectIdleTimeout = options.disconnectIdleTimeout || 5000;
        this.dropNodeConnectionAfter = options.dropNodeConnectionAfter || 5000;

        if (!options.duplicated) {
            this.duplicated = (msg: Message): boolean => {
                return false;
            };
        } else {
            this.duplicated = options.duplicated;
        }

        // socket id per browser tab
        let sid = this.getSession("chain:sid");

        const newSid = () => {
            sid = (Math.random() + 1).toString(36).substring(7);
            this.storeSession("chain:sid", sid);
        }

        if (sid == null) {
            newSid();
        }

        const newTransport = () => {
            for (let i = 0; i < this.transportConfig.length; i++) {
                const config = this.transportConfig[i];
                if (Transports[config.name]) {
                    this.transport = new Transports[config.name]({
                        ...config,
                        sid: sid,
                    });
                    break;
                }
            }

            // @TODO: LongPoll fallback

            this.transport.on("open", this.onTransportOpen.bind(this));
            this.transport.on("error", this.onTransportError.bind(this));
            this.transport.on("message", this.onTransportMessage.bind(this));
            this.transport.on("close", this.onTransportClose.bind(this));
        }

        let nodes: Array<Node> = [];

        const initTransport = (node: Node) => {
            if (this.transport) {
                if (this.transport.endpoint() == node.endpoint) {
                    return
                }
                this.transportListOld.push(this.transport);
                newSid();
            }

            this.endpoint = node.endpoint;

            const transport = this.transportListOld.find(transport => {
                return transport.endpoint() == node.endpoint;
            });
            if (transport) {
                this.transportListOld.splice(this.transportListOld.indexOf(transport), 1);
                this.transport = transport;
            } else {
                newTransport();
            }

            this.state = SocketStateEnum.CONNECTING;
            this.transport.connect(this.endpoint);
        }

        const tryConnectToNextNode = () => {
            let next: Node;
            for (let i = 0; i < nodes.length; i++) {
                const node = nodes[i];
                if (node.retry.tries() == 0) {
                    next = node;
                    break
                }

                if (!next || next.retry.tries() > node.retry.tries()) {
                    next = node;
                }
            }

            next.retry.retry();
        }

        const doGetNodes = async () => {
            let endpoints = await options.getNodes();
            if (!endpoints || endpoints.length === 0) {
                log(SOCKET, 'No nodes available for connection"');
                return;
            }

            let currentNode: Node;
            let newNodes: Array<Node> = [];
            endpoints.forEach(endpoint => {
                let node = nodes.find(node => node.endpoint == endpoint);
                if (node) {
                    node.retry.reset();
                } else {
                    node = {
                        endpoint: endpoint,
                        retry: new Retry((retry) => {
                            initTransport(node);
                        }, [1, 500, 1000, 2000, 5000])
                    }
                }
                newNodes.push(node);
                if (this.endpoint == endpoint) {
                    currentNode = node;
                }
            });

            nodes = newNodes;

            if (currentNode == nodes[0]) {
                // nothing to do, already connected to the priority server
                return
            }

            // Not connected to a listed server or currently connected to a fallback auxiliary server
            // Attempts to switch to the first server in the list
            return tryConnectToNextNode();
        }

        setTimeout(doGetNodes);
        setInterval(doGetNodes, (options.getNodesInterval || 30) * 1000);
    }

    getSession(key: string) { return this.sessionStorage.getItem(key) }

    storeSession(key: string, value: string) { this.sessionStorage.setItem(key, value) }

    getTimeout() {
        return this.timeout;
    }

    getRejoinInterval() {
        return this.rejoinInterval;
    }

    isConnected() {
        return this.state == SocketStateEnum.CONNECTED;
    }

    connect() {
        if (this.state == SocketStateEnum.CONNECTED || this.state == SocketStateEnum.CONNECTING) {
            return
        }
        this.state = SocketStateEnum.CONNECTING;
        if (this.transport) {
            this.transport.connect(this.endpoint);
        }
    }

    disconnect() {
        if (this.state == SocketStateEnum.DISCONNECTED || this.state == SocketStateEnum.DISCONNECTING || this.state == SocketStateEnum.ERRORED) {
            return
        }
        this.state = SocketStateEnum.DISCONNECTING;
        if (this.transport) {
            this.transport.close();
        } else {
            this.state = SocketStateEnum.DISCONNECTED;
        }
    }

    /**
     * Initiates a new channel for the given topic
     * 
     * @param topic 
     * @param params 
     * @param options 
     * @returns 
     */
    channel(topic: string, params: any = {}, options: ChannelOptions = {}): Channel {
        if (!this.isConnected()) {
            this.connect();
        }
        let channel = new Channel(topic, params, this, options);
        this.channels.push(channel);
        if (this.disconnectIdleTimer) {
            clearTimeout(this.disconnectIdleTimer);
        }
        return channel;
    }

    remove(channel: Channel) {
        let idx = this.channels.indexOf(channel);
        if (idx >= 0) {
            this.channels.splice(idx, 1);
            if (this.channels.length == 0) {
                this.disconnectIdleTimer = setTimeout(() => {
                    this.disconnect();
                    this.disconnectIdleTimer = undefined;
                }, this.disconnectIdleTimeout);
            }
        }
    }

    leave(topic: string) {
        let channel = this.channels.find(chn => {
            return chn.getTopic() === topic && (chn.isJoined() || chn.isJoining());
        })
        if (channel) {
            log(SOCKET, 'leaving topic "%s"', topic);
            channel.leave();
        }
    }

    push(message: Message, transport?: Transport) {
        let { topic, event, payload, ref, joinRef } = message;
        const data = encode(message);
        if (transport) {
            // _leave command only
            log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload);
            transport.send(data);
        } else if (this.state == SocketStateEnum.CONNECTED) {
            log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload);
            this.transport.send(data);
        } else {
            log(SOCKET, 'push %s %s (%s, %s) [scheduled]', topic, event, joinRef, ref, payload);
            this.sendBuffer.push(() => {
                log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload);
                this.transport.send(data);
            });
        }
    }

    /**
     * Return the next message ref, accounting for overflows
     * 
     * @returns 
     */
    nextRef() {
        this.ref++;
        if (this.ref === Number.MAX_SAFE_INTEGER) {
            this.ref = 1;
        }

        return this.ref;
    }

    private onTransportOpen(transport: Transport) {
        if (transport == this.transport) {
            log(SOCKET, 'connected to %s', this.endpoint);

            this.state = SocketStateEnum.CONNECTED;

            if (this.sendBuffer.length > 0) {
                // flush send buffer
                this.sendBuffer.forEach(callback => callback());
                this.sendBuffer.splice(0);
            }

            this.emit('open', transport);

            if (this.transportListOld.length > 0) {
                let old = this.transportListOld;
                this.transportListOld = [];
                setTimeout(() => {
                    old.forEach(transport => {
                        transport.close();
                    })
                }, this.dropNodeConnectionAfter);
            }
        }
    }

    private onTransportClose(transport: Transport, event: any) {
        if (transport == this.transport) {
            log(SOCKET, 'closed');
            this.state = SocketStateEnum.DISCONNECTED;
            this.emit('close');
        } else {
            let idx = this.transportListOld.indexOf(transport);
            if (idx >= 0) {
                this.transportListOld.splice(idx, 1);
            }
        }
    }

    private onTransportMessage(transport: Transport, data: string) {
        let message = decode(data);
        let { topic, event, payload, ref, joinRef } = message;
        log(SOCKET, 'receive %s %s %s',
            topic || '', event || '', (ref || joinRef) ? (`(${joinRef || ''}, ${ref || ''})`) : '', payload
        );

        // deduplication
        if (this.duplicated(message)) {
            this.emit('message:duplicated', message);
        } else {
            this.channels.forEach(channel => {
                channel.trigger(event, payload, topic, ref, joinRef)
            });
            this.emit('message', message);
        }
    }

    private onTransportError(transport: Transport, error: any) {
        if (transport == this.transport) {
            this.state = SocketStateEnum.ERRORED;
            this.emit('error', error);
        }
    }
}

export class Channel extends Events<Channel> {

    private topic: string;
    private socket: Socket
    private state: ChannelStateEnum;
    private timeout: number;
    private joinPush: Push;
    private joinedOnce: boolean;
    private rejoinRetry: Retry;
    private joinTransport: Transport;
    private readonly pushBuffer: Push[] = [];

    // Overridable message hook
    // Receives all events for specialized message handling before dispatching to the channel callbacks.
    // Must return the payload, modified or unmodified
    private onMessage: (event: string, payload: any, ref?: number, p_joinRef?: number) => any;

    constructor(topic: string, params: any, socket: Socket, options: ChannelOptions) {
        super();

        if (topic.includes(',') || topic.includes('*')) {
            throw new Error("Commas and asterisks are not allowed in topic names")
        }

        this.topic = topic;
        this.state = ChannelStateEnum.CLOSED;
        this.socket = socket;
        this.timeout = socket.getTimeout();
        this.onMessage = options.onMessage ? options.onMessage : (_e: string, payload: any) => payload;

        this.rejoinRetry = new Retry(() => {
            if (socket.isConnected()) {
                this.rejoin();
            }
        }, socket.getRejoinInterval());

        const onSocketError = () => {
            this.state = ChannelStateEnum.ERRORED;
            this.rejoinRetry.reset();
        };
        socket.on('error', onSocketError)

        let lastEndpoint: string;

        const onSocketOpen = (transport: Transport) => {
            this.rejoinRetry.reset();
            if (this.isErrored() || (this.joinTransport && (transport != this.joinTransport || transport.endpoint() != lastEndpoint))) {
                this.rejoin();
            }
            this.joinTransport = transport;
            lastEndpoint = transport.endpoint();
        }
        socket.on('open', onSocketOpen);

        this.joinPush = new Push(socket, this, '_join', params, this.timeout)
            .on('ok', () => {
                this.state = ChannelStateEnum.JOINED;
                this.rejoinRetry.reset();
                this.pushBuffer.forEach(push => push.send());
                this.pushBuffer.splice(0);
                this.emit('join:ok');
            })
            .on('error', () => {
                this.state = ChannelStateEnum.ERRORED;
                if (socket.isConnected()) {
                    this.rejoinRetry.retry();
                }
                this.emit('join:error');
            })
            .on('timeout', () => {
                log(CHANNEL, 'timeout %s (%s)', topic, this.getJoinRef(), this.joinPush.getTimeout());

                // leave (if joined on server)
                new Push(socket, this, '_leave', {}, this.timeout).send();

                this.state = ChannelStateEnum.ERRORED;
                this.joinPush.reset();
                if (socket.isConnected()) {
                    this.rejoinRetry.retry();
                }
                this.emit('join:timeout');
            });

        this.onClose(() => {
            if (this.isClosed()) {
                return;
            }
            log(CHANNEL, 'close %s %s', topic, this.getJoinRef());

            socket.off('open', onSocketOpen);
            socket.off('error', onSocketError)
            this.rejoinRetry.reset();
            this.state = ChannelStateEnum.CLOSED;
            socket.remove(this);
        });

        this.onError(reason => {
            log(CHANNEL, 'error %s', topic, reason);

            if (this.isJoining()) {
                this.joinPush.reset();
            }
            this.state = ChannelStateEnum.ERRORED;
            if (socket.isConnected()) {
                this.rejoinRetry.retry();
            }
        })

        this.on('_reply', (payload, ref) => {
            this.trigger(`chan_reply_${ref}`, payload);
        });
    }

    getTopic(): string {
        return this.topic;
    }

    getJoinRef() {
        return this.joinPush.getRef();
    }

    /**
     * Join the channel
     * 
     * @param timeout 
     * @returns 
     */
    join(timeout = this.timeout): Channel {
        if (this.joinedOnce) {
            throw new Error("tried to join multiple times. 'join' can only be called a single time per channel instance");
        } else {
            this.timeout = timeout;
            this.joinedOnce = true;
            this.rejoin();
            return this;
        }
    }

    private rejoin() {
        if (this.isLeaving()) {
            return;
        }

        // preciso salvar o transporte que fez o join
        this.socket.leave(this.topic);
        this.state = ChannelStateEnum.JOINING;
        this.joinPush.resend(this.timeout);
    }

    /**
      * Leaves the channel
      *
      * Unsubscribes from server events, and instructs channel to terminate on server
      *
      * Triggers onClose() hooks
      *
      * To receive leave acknowledgements, use the `receive` hook to bind to the server ack, ie:
      *
      * @example
      * channel.leave().on("ok", () => alert("left!") )
      * 
      * @param timeout 
      * @returns 
      */
    leave(timeout = this.timeout): Push {
        this.rejoinRetry.reset();
        this.joinPush.cancelTimeout();

        this.state = ChannelStateEnum.LEAVING;

        const onClose = () => {
            if (this.state == ChannelStateEnum.LEAVING) {
                log(CHANNEL, 'leave %s', this.topic);
                this.trigger('_close', 'leave');
            }
        }

        const leavePush = new Push(this.socket, this, '_leave', {}, timeout, this.joinTransport)
            .on('ok', onClose)
            .on('timeout', onClose);

        leavePush.send();

        if (!this.canPush()) {
            queueMicrotask(() => {
                leavePush.trigger('ok', {});
            });
        }

        return leavePush;
    }

    /**
      * Sends a message `event` to syntax with the `payload`.
      *
      * Syntax receives this in the `handle_in(event, payload, socket)` function. if syntax replies or it times out
      * (default 10000ms), then optionally the reply can be received.
      *
      * @example
      *  channel.push("event")
      *    .on("ok", payload => console.log("syntax replied:", payload))
      *    .on("error", err => console.log("syntax errored", err))
      *    .on("timeout", () => console.log("timed out pushing"))
      * 
      * @param event 
      * @param payload 
      * @param timeout 
      * @returns 
      */
    push(event: string, payload: any, timeout = this.timeout): Push {
        if (event.includes(',')) {
            throw new Error("Commas are not allowed in event");
        }

        payload = payload || {};
        if (!this.joinedOnce) {
            throw new Error(`tried to push '${event}' to '${this.topic}' before joining. Use channel.join() before pushing events`);
        }

        let push = new Push(this.socket, this, event, payload, timeout);
        if (this.canPush()) {
            push.send();
        } else {
            push.startTimeout();
            this.pushBuffer.push(push);
        }
        return push;
    }

    trigger(event: string, payload: any, p_topic?: string, ref?: number, p_joinRef?: number) {
        if (p_topic && this.topic !== p_topic) {
            // to other channel
            return;
        }
        if (p_joinRef && p_joinRef !== this.getJoinRef()) {
            // outdated message or to other channel
            return;
        }

        let handledPayload = this.onMessage(event, payload, ref, p_joinRef);
        if (payload && !handledPayload) {
            throw new Error("channel onMessage callbacks must return the payload, modified or unmodified");
        }

        this.emit(event, handledPayload, ref, p_joinRef || this.getJoinRef());
    }

    canPush() {
        return this.socket.isConnected() && this.isJoined();
    }

    onClose(callback: (...args: any) => void): () => void {
        this.on("_close", callback);
        return this.off.bind(this, "_close", callback);
    }

    onError(callback: (...args: any) => void): () => void {
        this.on("_error", callback);
        return this.off.bind(this, "_error", callback);
    }

    isClosed() {
        return this.state === ChannelStateEnum.CLOSED;
    }

    isErrored() {
        return this.state === ChannelStateEnum.ERRORED;
    }

    isJoined() {
        return this.state === ChannelStateEnum.JOINED;
    }

    isJoining() {
        return this.state === ChannelStateEnum.JOINING;
    }

    isLeaving() {
        return this.state === ChannelStateEnum.LEAVING;
    }
}

/**
 * a Push event
 */
export class Push {

    private ref?: number;
    private sent: boolean;
    private timer: any;
    private timeout: number;
    private socket: Socket;
    private channel: Channel;
    private received: any;
    private refEvent?: string;
    private transport?: Transport; // for _leave only
    private readonly event: string;
    private readonly events = new Events();
    private readonly payload: any;

    constructor(socket: Socket, channel: Channel, event: string, payload: any, timeout: number, transport?: Transport) {
        this.ref = socket.nextRef()
        this.event = event;
        this.payload = payload || {};
        this.timeout = timeout;
        this.socket = socket;
        this.channel = channel;
        this.transport = transport;
    }

    getRef() {
        return this.ref
    }

    getTimeout() {
        return this.timeout
    }

    on(event: string, callback: (...args: any) => void): Push {
        if (this.hasReceived(event)) {
            queueMicrotask(callback.bind(null, this.received.response));
        } else {
            this.events.on(event, callback);
        }
        return this;
    }

    send() {
        if (this.hasReceived('timeout')) {
            return;
        }
        this.startTimeout();
        this.sent = true;
        this.socket.push({
            ref: this.ref!,
            joinRef: this.channel.getJoinRef()!,
            topic: this.channel.getTopic(),
            event: this.event,
            payload: this.payload,
            payload_raw: '',
        }, this.transport);
    }

    resend(timeout: number) {
        this.timeout = timeout;
        this.reset();
        this.send();
    }

    reset() {
        this.channel.off(this.refEvent, this.onRefEventCallback);
        this.ref = undefined;
        this.sent = false;
        this.refEvent = undefined;
        this.received = null;
    }

    startTimeout() {
        if (this.timer) {
            this.cancelTimeout();
        }
        this.ref = this.socket.nextRef();
        this.refEvent = `chan_reply_${this.ref}`;

        this.channel.on(this.refEvent, this.onRefEventCallback);

        this.timer = setTimeout(() => {
            this.trigger("timeout", {});
        }, this.timeout);
    }

    cancelTimeout() {
        clearTimeout(this.timer);
        this.timer = null;
    }

    private onRefEventCallback = (payload: any) => {
        this.channel.off(this.refEvent, this.onRefEventCallback);
        this.cancelTimeout();
        this.received = payload;
        let { status, response, _ref } = payload;
        this.events.emit(status, response);
    }

    hasReceived(status: string) {
        return this.received && this.received.status === status;
    }

    trigger(status: string, response: any) {
        // event, payload, topic, ref, joinRef
        this.channel.trigger(this.refEvent!, { status, response });
    }
}

export interface Transport extends Events<Transport> {
    send(data: any): void;
    endpoint(): string;
    connect(endpoint: string): void;
    close(): void;
}

export interface TransportConstructor {
    new(options: TransportConfig): Transport;
}

/**
 * Channel transport using server-sent events
 */
export class TransportSSE extends Events<Transport> implements Transport {

    private source: EventSource;
    private endpointRaw: string;
    private endpointPush: string;
    private endpointEvents: string;
    private readonly options: TransportConfig;

    constructor(options: TransportConfig) {
        super();
        this.options = options;
    }

    send(data: any) {
        // fire and forget
        fetch(this.endpointPush, {
            ...(this.options.cors ? { mode: 'cors', credentials: 'include' } : {}),
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: data,
        }).catch((error) => {
            log(TRANSPORT, 'send error', error, data);
        });
    }

    endpoint(): string {
        return this.endpointRaw;
    }

    connect(endpoint: string) {
        if (this.source && this.endpointRaw == endpoint) {
            // avoid duplication
            return
        }
        if (this.source) {
            this.source.close();
        }

        this.endpointRaw = endpoint;
        this.endpointPush = parseUrl(endpoint, '/sse', { sid: this.options.sid });
        this.endpointEvents = parseUrl(endpoint, '/sse', { ...(this.options.params), sid: this.options.sid });

        this.source = new EventSource(this.endpointEvents, {
            ...(this.options.cors ? { withCredentials: true } : {}),
        });

        this.source.onmessage = (event) => {
            log(TRANSPORT, 'message', event);
            this.emit('message', this, event.data);
        };

        this.source.onerror = (event) => {
            log(TRANSPORT, 'error', event);
            this.emit('error', this);
        };

        this.source.onopen = (event) => {
            log(TRANSPORT, 'open', event);
            this.emit('open', this);
        };
    }

    close() {
        if (!this.source) {
            return;
        }
        log(TRANSPORT, 'close');
        this.source.close();
        this.emit('close', this);
        this.source = null;
    }
}


export const Transports: { [key: string]: TransportConstructor } = {
    "SSE": TransportSSE
}

/**
 * Timer to retry callback
 */
export class Retry {
    private _tries = 0;
    private timeout: any;
    private readonly callback: (retry: number) => void;
    private readonly intervals: number[];
    private readonly intervalMax: number;

    constructor(callback: (retry: number) => void, intervals: number[]) {
        this.callback = callback;
        this.intervals = intervals.slice(0).sort();
        this.intervalMax = Math.max(...this.intervals);
    }

    tries(): number {
        return this._tries
    }

    lastInterval(): number {
        return this.intervals[this._tries] || this.intervalMax;
    }

    reset() {
        this._tries = 0;
        clearTimeout(this.timeout);
    }

    retry() {
        clearTimeout(this.timeout);
        this.timeout = setTimeout(() => {
            this._tries++;
            this.callback(this._tries);
        }, this.intervals[this._tries] || this.intervalMax);
    }
}

function parseUrl(endpoint: string, suffix: string, params?: { [key: string]: any } | undefined) {

    let isHttp = endpoint.startsWith('http://');
    let isHttps = endpoint.startsWith('https://');
    let isProtocol = endpoint.startsWith('//');

    if (isHttps) {
        endpoint = endpoint.replace('https://', '');
    } else if (isHttp) {
        endpoint = endpoint.replace('http://', '');
    } else if (isProtocol) {
        endpoint = endpoint.replace('//', '');
    }

    let parts = endpoint.split('?');
    let basePath = (parts[0] + suffix).replaceAll(/[/]+/g, '/');
    let queryString = parts[1] || '';
    if (params) {
        for (let key in params) {
            if (queryString.length > 0) {
                queryString = queryString + '&';
            }
            queryString = queryString + `${key}=${encodeURIComponent(params[key])}`;
        }
    }

    if (isHttps) {
        basePath = `https://${basePath}`;
    } else if (isHttp) {
        basePath = `http://${basePath}`;
    } else if (isProtocol) {
        basePath = `//${basePath}`;
    }

    if (queryString != '') {
        return `${basePath}?${queryString}`;
    }

    return basePath;
}

export function encode(message: Message): string {
    let { joinRef, ref, topic, event, payload } = message;
    return JSON.stringify([MessageKindEnum.PUSH, joinRef, ref, topic, event, payload]);
}

export function decode(rawMessage: string): Message {
    // Push      = [kind, joinRef, ref,  topic, event, payload]
    // Reply     = [kind, joinRef, ref, status,        payload]
    // Broadcast = [kind,                topic, event, payload]    
    let [kind, joinRef, ref, topic, event, payload] = JSON.parse(rawMessage);
    let countParts = 5;
    if (kind === MessageKindEnum.REPLY) {
        countParts = 4;
        payload = { status: topic === 0 ? 'ok' : 'error', response: event };
        event = '_reply';
        topic = undefined;
    } else if (kind === MessageKindEnum.BROADCAST) {
        countParts = 3;
        payload = topic;
        event = ref;
        topic = joinRef;
        joinRef = ref = undefined;
    }

    // extract raw payload    
    let payload_raw = '';
    let lastIndexOfComma = -1;
    for (let i = 0; i < countParts; i++) {
        lastIndexOfComma = rawMessage.indexOf(',', lastIndexOfComma + 1);
        if (lastIndexOfComma == -1) {
            break
        }
    }
    if (lastIndexOfComma != - 1) {
        payload_raw = rawMessage.substring(lastIndexOfComma + 1, rawMessage.length - 1);
    }

    return { joinRef, ref, topic, event, payload, payload_raw, kind };
}

let logGroupLen = Math.max(TRANSPORT.length, CHANNEL.length, SOCKET.length);

export function log(group: string, template: string, ...params: any) {
    if (typeof group === 'string' && Options[`Debug${group}`] === false) {
        return;
    }
    if (Options.Debug || (typeof group === 'string' && Options[`Debug${group}`])) {
        if (typeof template !== 'string') {
            params = [template, ...params];
            template = '';

            if (typeof group !== 'string') {
                params = [group, ...params];
                group = '';
            }
        }

        logGroupLen = Math.max(logGroupLen, group.length);
        console.log(
            `${`                           ${group}`.substr(-logGroupLen)}: ${template}`,
            ...params
        );
    }
}

/**
 * Structure used for deduplication algorithms
 * 
 * Example:
 *      const history = new History(5);
 *      let socket = new chain.Socket({
 *          duplicated: (msg: Message) => { return history.exists(msg.payload.messageId) }
 *      });
 */
export class History {
    private readonly history: any = {};
    private readonly ttlSeconds: number;

    constructor(ttlSeconds?: number) {
        this.ttlSeconds = Math.max(Math.round((ttlSeconds || 5)), 1);
        setInterval(this.rotate.bind(this), 1000);
    }

    /**
     * Checks if the given value exists in the history.
     * 
     * @param key 
     * @param value 
     * @returns 
     */
    exists(key: string): boolean {

        let exists = false;

        for (let i = 0; i < this.ttlSeconds; i++) {
            let slice = this.history[`_${i}`];
            if (slice && slice[key]) {
                exists = true;
                break
            }
        }

        if (!this.history["_0"]) {
            this.history["_0"] = {};
        }
        this.history["_0"][key] = true;

        return exists;
    }

    /**
     * Rotates the message history.
     */
    private rotate() {

        /*
        every second:
            lastMessagesForDedup = {
                "5": {}, // will be removed
                "4": {}, // will be renamed to "5" (removing "5")
                "3": {}, // will be renamed to "4"
                "2": {}, // will be renamed to "3"      
                "1": {}, // will be renamed to "2"
                "0": {}, // will be renamed to "1"
                // "0" = {} .. will be created (on next defaultDuplicated call)                    
            }
        */
        for (let i = this.ttlSeconds; i > 0; i--) {
            let secKey = `_${i}`;
            if (this.history[secKey]) {
                if (i == this.ttlSeconds) { // last
                    delete this.history[secKey];
                } else {
                    // move
                }
            }
        }
    }
}
