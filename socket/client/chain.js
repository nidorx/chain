/**
 * Client do serviço de mensageria do papo
 *
 * Versão modificada do https://github.com/nidorx/chain/tree/main/socket/client
 */
const SOCKET = 'Socket';
const CHANNEL = 'Channel';
const TRANSPORT = 'Transport';
var MessageKindEnum;
(function (MessageKindEnum) {
    MessageKindEnum[MessageKindEnum["PUSH"] = 0] = "PUSH";
    MessageKindEnum[MessageKindEnum["REPLY"] = 1] = "REPLY";
    MessageKindEnum[MessageKindEnum["BROADCAST"] = 2] = "BROADCAST";
})(MessageKindEnum || (MessageKindEnum = {}));
var ChannelStateEnum;
(function (ChannelStateEnum) {
    ChannelStateEnum[ChannelStateEnum["CLOSED"] = 0] = "CLOSED";
    ChannelStateEnum[ChannelStateEnum["ERRORED"] = 1] = "ERRORED";
    ChannelStateEnum[ChannelStateEnum["JOINED"] = 2] = "JOINED";
    ChannelStateEnum[ChannelStateEnum["JOINING"] = 3] = "JOINING";
    ChannelStateEnum[ChannelStateEnum["LEAVING"] = 4] = "LEAVING";
})(ChannelStateEnum || (ChannelStateEnum = {}));
/**
 * Copyright 2016 Andrey Sitnik <andrey@sitnik.ru>, https://github.com/ai/nanoevents/blob/main/LICENSE
 */
class Events {
    events = {};
    emit(event, ...args) {
        let callbacks = this.events[event] || [];
        for (let i = 0, length = callbacks.length; i < length; i++) {
            callbacks[i](...args);
        }
    }
    on(event, callback) {
        this.events[event]?.push(callback) || (this.events[event] = [callback]);
        // off
        return () => {
            let callbacks = this.events[event];
            if (callbacks) {
                let idx = callbacks.indexOf(callback);
                if (idx >= 0) {
                    callbacks.splice(idx, 1);
                }
            }
        };
    }
}
var SocketStateEnum;
(function (SocketStateEnum) {
    SocketStateEnum[SocketStateEnum["DISCONNECTED"] = 0] = "DISCONNECTED";
    SocketStateEnum[SocketStateEnum["CONNECTING"] = 2] = "CONNECTING";
    SocketStateEnum[SocketStateEnum["CONNECTED"] = 3] = "CONNECTED";
    SocketStateEnum[SocketStateEnum["DISCONNECTING"] = 4] = "DISCONNECTING";
    SocketStateEnum[SocketStateEnum["ERRORED"] = 5] = "ERRORED";
})(SocketStateEnum || (SocketStateEnum = {}));
/**
 * Interface de comunicação com o servidor
 */
export class Socket extends Events {
    ref = 1;
    state;
    connected = false;
    disconnectIdleTimer;
    channels = [];
    sendBuffer = [];
    transport;
    timeout;
    endpoint;
    sessionStorage;
    rejoinInterval;
    reconnectInterval;
    disconnectIdleTimeout;
    constructor(endpoint, options = {}) {
        super();
        this.state = SocketStateEnum.DISCONNECTED;
        this.endpoint = endpoint;
        this.timeout = options.timeout || 30000;
        this.sessionStorage = options.sessionStorage || (window.sessionStorage);
        this.rejoinInterval = options.rejoinInterval || [1000, 2000, 5000, 10000];
        this.reconnectInterval = options.reconnectInterval || [10, 50, 100, 150, 200, 250, 500, 1000, 2000, 5000];
        this.disconnectIdleTimeout = options.disconnectIdleTimeout || 5000;
        // socket id per browser tab
        let sid = this.getSession("chain:sid");
        if (sid == null) {
            sid = (Math.random() + 1).toString(36).substring(7);
            this.storeSession("chain:sid", sid);
        }
        this.transport = new (options.transport || TransportSSE)(endpoint, {
            ...(options.transportOptions || {}),
            sid: sid,
        });
        this.transport.on("open", this.onConnOpen.bind(this));
        this.transport.on("error", this.onConnError.bind(this));
        this.transport.on("message", this.onConnMessage.bind(this));
        this.transport.on("close", this.onConnClose.bind(this));
    }
    getSession(key) { return this.sessionStorage.getItem(key); }
    storeSession(key, value) { this.sessionStorage.setItem(key, value); }
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
            return;
        }
        this.state = SocketStateEnum.CONNECTING;
        return this.transport.connect();
    }
    disconnect() {
        if (this.state == SocketStateEnum.DISCONNECTED || this.state == SocketStateEnum.DISCONNECTING || this.state == SocketStateEnum.ERRORED) {
            return;
        }
        this.state = SocketStateEnum.DISCONNECTING;
        this.transport.close();
    }
    /**
     * Initiates a new channel for the given topic
     *
     * @param topic
     * @param params
     * @param options
     * @returns
     */
    channel(topic, params = {}, options = {}) {
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
    remove(channel) {
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
    leaveOpenTopic(topic) {
        let dupChannel = this.channels.find(channel => {
            return channel.getTopic() === topic && (channel.isJoined() || channel.isJoining());
        });
        if (dupChannel) {
            log(SOCKET, 'leaving duplicate topic "%s"', topic);
            dupChannel.leave();
        }
    }
    push(message) {
        let { topic, event, payload, ref, joinRef } = message;
        const data = encode(message);
        if (this.state == SocketStateEnum.CONNECTED) {
            log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload);
            this.transport.send(data);
        }
        else {
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
    onConnOpen() {
        log(SOCKET, 'connected to %s', this.endpoint);
        this.state = SocketStateEnum.CONNECTED;
        if (this.sendBuffer.length > 0) {
            // flush send buffer
            this.sendBuffer.forEach(callback => callback());
            this.sendBuffer.splice(0);
        }
        this.emit('open');
    }
    onConnClose(event) {
        this.state = SocketStateEnum.DISCONNECTED;
        this.emit('close');
    }
    onConnMessage(data) {
        let message = decode(data);
        let { topic, event, payload, ref, joinRef } = message;
        log(SOCKET, 'receive %s %s %s', topic || '', event || '', (ref || joinRef) ? (`(${joinRef || ''}, ${ref || ''})`) : '', payload);
        this.channels.forEach(channel => {
            channel.trigger(event, payload, topic, ref, joinRef);
        });
        this.emit('message', message);
    }
    onConnError(error) {
        this.state = SocketStateEnum.ERRORED;
        this.emit('error', error);
    }
}
export class Channel extends Events {
    topic;
    socket;
    state;
    timeout;
    joinPush;
    joinedOnce;
    rejoinRetry;
    pushBuffer = [];
    // Overridable message hook
    // Receives all events for specialized message handling before dispatching to the channel callbacks.
    // Must return the payload, modified or unmodified
    onMessage;
    constructor(topic, params, socket, options) {
        super();
        this.topic = topic;
        this.socket = socket;
        this.timeout = socket.getTimeout();
        this.state = ChannelStateEnum.CLOSED;
        this.onMessage = options.onMessage ? options.onMessage : (_e, payload) => payload;
        this.rejoinRetry = new Retry(() => {
            if (socket.isConnected()) {
                this.rejoin();
            }
        }, socket.getRejoinInterval());
        const cancelOnSocketError = socket.on('error', () => {
            this.rejoinRetry.reset();
        });
        const cancelOnSocketOpen = socket.on('open', () => {
            this.rejoinRetry.reset();
            if (this.isErrored()) {
                this.rejoin();
            }
        });
        this.joinPush = new Push(socket, this, '_join', params, this.timeout)
            .on('ok', () => {
            this.state = ChannelStateEnum.JOINED;
            this.rejoinRetry.reset();
            this.pushBuffer.forEach(push => push.send());
            this.pushBuffer.splice(0);
        })
            .on('error', () => {
            this.state = ChannelStateEnum.ERRORED;
            if (socket.isConnected()) {
                this.rejoinRetry.retry();
            }
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
        });
        this.onClose(() => {
            if (this.isClosed()) {
                return;
            }
            log(CHANNEL, 'close %s %s', topic, this.getJoinRef());
            cancelOnSocketOpen();
            cancelOnSocketError();
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
        });
        this.on('_reply', (payload, ref) => {
            this.trigger(`chan_reply_${ref}`, payload);
        });
    }
    getTopic() {
        return this.topic;
    }
    getJoinRef() {
        return this.joinPush.getRef();
    }
    /**
     * Join the channel
     *
     * @param p_timeout
     * @returns
     */
    join(p_timeout = this.timeout) {
        if (this.joinedOnce) {
            throw new Error("tried to join multiple times. 'join' can only be called a single time per channel instance");
        }
        else {
            this.timeout = p_timeout;
            this.joinedOnce = true;
            this.rejoin();
            return this.joinPush;
        }
    }
    rejoin() {
        if (this.isLeaving()) {
            return;
        }
        this.socket.leaveOpenTopic(this.topic);
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
      * @param p_timeout
      * @returns
      */
    leave(p_timeout = this.timeout) {
        this.rejoinRetry.reset();
        this.joinPush.cancelTimeout();
        this.state = ChannelStateEnum.LEAVING;
        let onClose = () => {
            log(CHANNEL, 'leave %s', this.topic);
            this.trigger('_close', 'leave');
        };
        let leavePush = new Push(this.socket, this, '_leave', {}, p_timeout)
            .on('ok', onClose)
            .on('timeout', onClose);
        leavePush.send();
        if (!this.canPush()) {
            leavePush.trigger('ok', {});
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
      * @param p_timeout
      * @returns
      */
    push(event, payload, p_timeout = this.timeout) {
        payload = payload || {};
        if (!this.joinedOnce) {
            throw new Error(`tried to push '${event}' to '${this.topic}' before joining. Use channel.join() before pushing events`);
        }
        let push = new Push(this.socket, this, event, payload, p_timeout);
        if (this.canPush()) {
            push.send();
        }
        else {
            push.startTimeout();
            this.pushBuffer.push(push);
        }
        return push;
    }
    trigger(event, payload, p_topic, ref, p_joinRef) {
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
    onClose(callback) {
        return this.on("_close", callback);
    }
    onError(callback) {
        return this.on("_error", callback);
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
    ref;
    sent;
    timer;
    timeout;
    socket;
    channel;
    received;
    refEvent;
    refEventCancel;
    event;
    events = new Events();
    payload;
    constructor(socket, channel, event, payload, timeout) {
        this.ref = socket.nextRef();
        this.event = event;
        this.payload = payload || {};
        this.timeout = timeout;
        this.socket = socket;
        this.channel = channel;
    }
    getRef() {
        return this.ref;
    }
    getTimeout() {
        return this.timeout;
    }
    on(event, callback) {
        if (this.hasReceived(event)) {
            queueMicrotask(callback.bind(null, this.received.response));
        }
        else {
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
            ref: this.ref,
            joinRef: this.channel.getJoinRef(),
            topic: this.channel.getTopic(),
            event: this.event,
            payload: this.payload,
        });
    }
    resend(p_timeout) {
        this.timeout = p_timeout;
        this.reset();
        this.send();
    }
    reset() {
        if (this.refEventCancel) {
            this.refEventCancel();
            this.refEventCancel = undefined;
        }
        this.ref = undefined;
        this.sent = false;
        this.refEvent = undefined;
        this.received = null;
    }
    cancelTimeout() {
        clearTimeout(this.timer);
        this.timer = null;
    }
    startTimeout() {
        if (this.timer) {
            this.cancelTimeout();
        }
        this.ref = this.socket.nextRef();
        this.refEvent = `chan_reply_${this.ref}`;
        this.refEventCancel = this.channel.on(this.refEvent, (payload) => {
            if (this.refEventCancel) {
                this.refEventCancel();
                this.refEventCancel = undefined;
            }
            this.cancelTimeout();
            this.received = payload;
            let { status, response, _ref } = payload;
            this.events.emit(status, response);
        });
        this.timer = setTimeout(() => {
            this.trigger("timeout", {});
        }, this.timeout);
    }
    hasReceived(status) {
        return this.received && this.received.status === status;
    }
    trigger(status, response) {
        // event, payload, topic, ref, joinRef
        this.channel.trigger(this.refEvent, { status, response });
    }
}
/**
 * Channel transport using server-sent events
 */
export class TransportSSE extends Events {
    source;
    options;
    endpoint;
    endpointPush;
    constructor(endpoint, options = {}) {
        super();
        this.options = options;
        this.endpoint = parseUrl(endpoint, '/sse', { ...(this.options.params), sid: options.sid });
        this.endpointPush = parseUrl(endpoint, '/sse', { sid: options.sid });
    }
    send(data) {
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
    connect() {
        this.source = new EventSource(this.endpoint, {
            ...(this.options.cors ? { withCredentials: true } : {}),
        });
        this.source.onmessage = (event) => {
            log(TRANSPORT, 'message', event);
            this.emit('message', event.data);
        };
        this.source.onerror = (event) => {
            log(TRANSPORT, 'error', event);
            this.emit('error');
        };
        this.source.onopen = (event) => {
            log(TRANSPORT, 'open', event);
            this.emit('open');
        };
    }
    close() {
        if (!this.source) {
            return;
        }
        log(TRANSPORT, 'close');
        this.source.close();
        this.emit('close');
    }
}
/**
 * Timer to retry callback
 */
export class Retry {
    tries = 0;
    timeout;
    callback;
    intervals;
    intervalMax;
    constructor(callback, intervals) {
        this.callback = callback;
        this.intervals = intervals.slice(0).sort();
        this.intervalMax = Math.max(...this.intervals);
    }
    reset() {
        this.tries = 0;
        clearTimeout(this.timeout);
    }
    retry() {
        clearTimeout(this.timeout);
        this.timeout = setTimeout(() => {
            this.tries++;
            this.callback();
        }, this.intervals[this.tries] || this.intervalMax);
    }
}
function parseUrl(endpoint, suffix, params) {
    let isHttp = endpoint.startsWith('http://');
    let isHttps = endpoint.startsWith('https://');
    let isProtocol = endpoint.startsWith('//');
    if (isHttps) {
        endpoint = endpoint.replace('https://', '');
    }
    else if (isHttp) {
        endpoint = endpoint.replace('http://', '');
    }
    else if (isProtocol) {
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
    }
    else if (isHttp) {
        basePath = `http://${basePath}`;
    }
    else if (isProtocol) {
        basePath = `//${basePath}`;
    }
    if (queryString != '') {
        return `${basePath}?${queryString}`;
    }
    return basePath;
}
export const Transport = { SSE: typeof TransportSSE };
export const Options = {
    Debug: false,
};
export function encode(message) {
    let { joinRef, ref, topic, event, payload } = message;
    let s = JSON.stringify([MessageKindEnum.PUSH, joinRef, ref, topic, event, payload]);
    return s.substr(1, s.length - 2);
}
export function decode(rawMessage) {
    // Push      = [kind, joinRef, ref,  topic, event, payload]
    // Reply     = [kind, joinRef, ref, status,        payload]
    // Broadcast = [kind,                topic, event, payload]
    let [kind, joinRef, ref, topic, event, payload] = JSON.parse(`[${rawMessage}]`);
    if (kind === MessageKindEnum.REPLY) {
        payload = { status: topic === 0 ? 'ok' : 'error', response: event };
        event = '_reply';
        topic = undefined;
    }
    else if (kind === MessageKindEnum.BROADCAST) {
        payload = topic;
        event = ref;
        topic = joinRef;
        joinRef = ref = undefined;
    }
    return { joinRef, ref, topic, event, payload, kind };
}
let logGroupLen = Math.max(TRANSPORT.length, CHANNEL.length, SOCKET.length);
export function log(group, template, ...params) {
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
        console.log(`${`                           ${group}`.substr(-logGroupLen)}: ${template}`, ...params);
    }
}
//# sourceMappingURL=chain.js.map