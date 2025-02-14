export const Options = {
    Debug: true,
};
export var MessageKindEnum;
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
var SocketStateEnum;
(function (SocketStateEnum) {
    SocketStateEnum[SocketStateEnum["DISCONNECTED"] = 0] = "DISCONNECTED";
    SocketStateEnum[SocketStateEnum["CONNECTING"] = 2] = "CONNECTING";
    SocketStateEnum[SocketStateEnum["CONNECTED"] = 3] = "CONNECTED";
    SocketStateEnum[SocketStateEnum["DISCONNECTING"] = 4] = "DISCONNECTING";
    SocketStateEnum[SocketStateEnum["ERRORED"] = 5] = "ERRORED";
})(SocketStateEnum || (SocketStateEnum = {}));
const SOCKET = 'Socket';
const CHANNEL = 'Channel';
const TRANSPORT = 'Transport';
export class Events {
    events = {};
    emit(event, ...args) {
        for (let callbacks = this.events[event] || [], i = 0, l = callbacks.length; i < l; i++) {
            callbacks[i](...args);
        }
    }
    on(event, callback) {
        this.events[event]?.push(callback) || (this.events[event] = [callback]);
        return this;
    }
    off(event, callback) {
        let callbacks = this.events[event];
        if (callbacks) {
            let idx = callbacks.indexOf(callback);
            if (idx >= 0) {
                callbacks.splice(idx, 1);
            }
        }
    }
}
;
export class Socket extends Events {
    ref = 1;
    state;
    endpoint;
    transport;
    transportListOld;
    disconnectIdleTimer;
    timeout;
    channels = [];
    duplicated;
    sendBuffer = [];
    sessionStorage;
    rejoinInterval;
    transportConfig;
    disconnectIdleTimeout;
    dropNodeConnectionAfter;
    constructor(options) {
        super();
        this.state = SocketStateEnum.DISCONNECTED;
        this.timeout = options.timeout || 30000;
        this.sessionStorage = options.sessionStorage || (window.sessionStorage);
        this.transportConfig = options.transport || [{ name: "SSE" }];
        this.rejoinInterval = options.rejoinInterval || [1000, 2000, 5000, 10000];
        this.transportListOld = [];
        this.disconnectIdleTimeout = options.disconnectIdleTimeout || 5000;
        this.dropNodeConnectionAfter = options.dropNodeConnectionAfter || 5000;
        if (!options.duplicated) {
            this.duplicated = (msg) => {
                return false;
            };
        }
        else {
            this.duplicated = options.duplicated;
        }
        let sid = this.getSession("chain:sid");
        const newSid = () => {
            sid = (Math.random() + 1).toString(36).substring(7);
            this.storeSession("chain:sid", sid);
        };
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
            this.transport.on("open", this.onTransportOpen.bind(this));
            this.transport.on("error", this.onTransportError.bind(this));
            this.transport.on("message", this.onTransportMessage.bind(this));
            this.transport.on("close", this.onTransportClose.bind(this));
        };
        let nodes = [];
        const initTransport = (node) => {
            if (this.transport) {
                if (this.transport.endpoint() == node.endpoint) {
                    return;
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
            }
            else {
                newTransport();
            }
            this.state = SocketStateEnum.CONNECTING;
            this.transport.connect(this.endpoint);
        };
        const tryConnectToNextNode = () => {
            let next;
            for (let i = 0; i < nodes.length; i++) {
                const node = nodes[i];
                if (node.retry.tries() == 0) {
                    next = node;
                    break;
                }
                if (!next || next.retry.tries() > node.retry.tries()) {
                    next = node;
                }
            }
            next.retry.retry();
        };
        const doGetNodes = async () => {
            let endpoints = await options.getNodes();
            if (!endpoints || endpoints.length === 0) {
                log(SOCKET, 'No nodes available for connection"');
                return;
            }
            let currentNode;
            let newNodes = [];
            endpoints.forEach(endpoint => {
                let node = nodes.find(node => node.endpoint == endpoint);
                if (node) {
                    node.retry.reset();
                }
                else {
                    node = {
                        endpoint: endpoint,
                        retry: new Retry((retry) => {
                            initTransport(node);
                        }, [1, 500, 1000, 2000, 5000])
                    };
                }
                newNodes.push(node);
                if (this.endpoint == endpoint) {
                    currentNode = node;
                }
            });
            nodes = newNodes;
            if (currentNode == nodes[0]) {
                return;
            }
            return tryConnectToNextNode();
        };
        setTimeout(doGetNodes);
        setInterval(doGetNodes, (options.getNodesInterval || 30) * 1000);
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
        if (this.transport) {
            this.transport.connect(this.endpoint);
        }
    }
    disconnect() {
        if (this.state == SocketStateEnum.DISCONNECTED || this.state == SocketStateEnum.DISCONNECTING || this.state == SocketStateEnum.ERRORED) {
            return;
        }
        this.state = SocketStateEnum.DISCONNECTING;
        if (this.transport) {
            this.transport.close();
        }
        else {
            this.state = SocketStateEnum.DISCONNECTED;
        }
    }
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
    leave(topic) {
        let channel = this.channels.find(chn => {
            return chn.getTopic() === topic && (chn.isJoined() || chn.isJoining());
        });
        if (channel) {
            log(SOCKET, 'leaving topic "%s"', topic);
            channel.leave();
        }
    }
    push(message, transport) {
        let { topic, event, payload, ref, joinRef } = message;
        const data = encode(message);
        if (transport) {
            log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload);
            transport.send(data);
        }
        else if (this.state == SocketStateEnum.CONNECTED) {
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
    nextRef() {
        this.ref++;
        if (this.ref === Number.MAX_SAFE_INTEGER) {
            this.ref = 1;
        }
        return this.ref;
    }
    onTransportOpen(transport) {
        if (transport == this.transport) {
            log(SOCKET, 'connected to %s', this.endpoint);
            this.state = SocketStateEnum.CONNECTED;
            if (this.sendBuffer.length > 0) {
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
                    });
                }, this.dropNodeConnectionAfter);
            }
        }
    }
    onTransportClose(transport, event) {
        if (transport == this.transport) {
            log(SOCKET, 'closed');
            this.state = SocketStateEnum.DISCONNECTED;
            this.emit('close');
        }
        else {
            let idx = this.transportListOld.indexOf(transport);
            if (idx >= 0) {
                this.transportListOld.splice(idx, 1);
            }
        }
    }
    onTransportMessage(transport, data) {
        let message = decode(data);
        let { topic, event, payload, ref, joinRef } = message;
        log(SOCKET, 'receive %s %s %s', topic || '', event || '', (ref || joinRef) ? (`(${joinRef || ''}, ${ref || ''})`) : '', payload);
        if (this.duplicated(message)) {
            this.emit('message:duplicated', message);
        }
        else {
            this.channels.forEach(channel => {
                channel.trigger(event, payload, topic, ref, joinRef);
            });
            this.emit('message', message);
        }
    }
    onTransportError(transport, error) {
        if (transport == this.transport) {
            this.state = SocketStateEnum.ERRORED;
            this.emit('error', error);
        }
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
    joinTransport;
    pushBuffer = [];
    onMessage;
    constructor(topic, params, socket, options) {
        super();
        if (topic.includes(',') || topic.includes('*')) {
            throw new Error("Commas and asterisks are not allowed in topic names");
        }
        this.topic = topic;
        this.state = ChannelStateEnum.CLOSED;
        this.socket = socket;
        this.timeout = socket.getTimeout();
        this.onMessage = options.onMessage ? options.onMessage : (_e, payload) => payload;
        this.rejoinRetry = new Retry(() => {
            if (socket.isConnected()) {
                this.rejoin();
            }
        }, socket.getRejoinInterval());
        const onSocketError = () => {
            this.state = ChannelStateEnum.ERRORED;
            this.rejoinRetry.reset();
        };
        socket.on('error', onSocketError);
        let lastEndpoint;
        const onSocketOpen = (transport) => {
            this.rejoinRetry.reset();
            if (this.isErrored() || (this.joinTransport && (transport != this.joinTransport || transport.endpoint() != lastEndpoint))) {
                this.rejoin();
            }
            this.joinTransport = transport;
            lastEndpoint = transport.endpoint();
        };
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
            socket.off('error', onSocketError);
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
    join(timeout = this.timeout) {
        if (this.joinedOnce) {
            throw new Error("tried to join multiple times. 'join' can only be called a single time per channel instance");
        }
        else {
            this.timeout = timeout;
            this.joinedOnce = true;
            this.rejoin();
            return this;
        }
    }
    rejoin() {
        if (this.isLeaving()) {
            return;
        }
        this.socket.leave(this.topic);
        this.state = ChannelStateEnum.JOINING;
        this.joinPush.resend(this.timeout);
    }
    leave(timeout = this.timeout) {
        this.rejoinRetry.reset();
        this.joinPush.cancelTimeout();
        this.state = ChannelStateEnum.LEAVING;
        const onClose = () => {
            if (this.state == ChannelStateEnum.LEAVING) {
                log(CHANNEL, 'leave %s', this.topic);
                this.trigger('_close', 'leave');
            }
        };
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
    push(event, payload, timeout = this.timeout) {
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
        }
        else {
            push.startTimeout();
            this.pushBuffer.push(push);
        }
        return push;
    }
    trigger(event, payload, p_topic, ref, p_joinRef) {
        if (p_topic && this.topic !== p_topic) {
            return;
        }
        if (p_joinRef && p_joinRef !== this.getJoinRef()) {
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
        this.on("_close", callback);
        return this.off.bind(this, "_close", callback);
    }
    onError(callback) {
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
export class Push {
    ref;
    sent;
    timer;
    timeout;
    socket;
    channel;
    received;
    refEvent;
    transport;
    event;
    events = new Events();
    payload;
    constructor(socket, channel, event, payload, timeout, transport) {
        this.ref = socket.nextRef();
        this.event = event;
        this.payload = payload || {};
        this.timeout = timeout;
        this.socket = socket;
        this.channel = channel;
        this.transport = transport;
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
            payload_raw: '',
        }, this.transport);
    }
    resend(timeout) {
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
    onRefEventCallback = (payload) => {
        this.channel.off(this.refEvent, this.onRefEventCallback);
        this.cancelTimeout();
        this.received = payload;
        let { status, response, _ref } = payload;
        this.events.emit(status, response);
    };
    hasReceived(status) {
        return this.received && this.received.status === status;
    }
    trigger(status, response) {
        this.channel.trigger(this.refEvent, { status, response });
    }
}
export class TransportSSE extends Events {
    source;
    endpointRaw;
    endpointPush;
    endpointEvents;
    options;
    constructor(options) {
        super();
        this.options = options;
    }
    send(data) {
        fetch(this.endpointPush, {
            ...(this.options.cors ? { mode: 'cors', credentials: 'include' } : {}),
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: data,
        }).catch((error) => {
            log(TRANSPORT, 'send error', error, data);
        });
    }
    endpoint() {
        return this.endpointRaw;
    }
    connect(endpoint) {
        if (this.source && this.endpointRaw == endpoint) {
            return;
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
export const Transports = {
    "SSE": TransportSSE
};
export class Retry {
    _tries = 0;
    timeout;
    callback;
    intervals;
    intervalMax;
    constructor(callback, intervals) {
        this.callback = callback;
        this.intervals = intervals.slice(0).sort();
        this.intervalMax = Math.max(...this.intervals);
    }
    tries() {
        return this._tries;
    }
    lastInterval() {
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
export function encode(message) {
    let { joinRef, ref, topic, event, payload } = message;
    return JSON.stringify([MessageKindEnum.PUSH, joinRef, ref, topic, event, payload]);
}
export function decode(rawMessage) {
    let [kind, joinRef, ref, topic, event, payload] = JSON.parse(rawMessage);
    let countParts = 5;
    if (kind === MessageKindEnum.REPLY) {
        countParts = 4;
        payload = { status: topic === 0 ? 'ok' : 'error', response: event };
        event = '_reply';
        topic = undefined;
    }
    else if (kind === MessageKindEnum.BROADCAST) {
        countParts = 3;
        payload = topic;
        event = ref;
        topic = joinRef;
        joinRef = ref = undefined;
    }
    let payload_raw = '';
    let lastIndexOfComma = -1;
    for (let i = 0; i < countParts; i++) {
        lastIndexOfComma = rawMessage.indexOf(',', lastIndexOfComma + 1);
        if (lastIndexOfComma == -1) {
            break;
        }
    }
    if (lastIndexOfComma != -1) {
        payload_raw = rawMessage.substring(lastIndexOfComma + 1, rawMessage.length - 1);
    }
    return { joinRef, ref, topic, event, payload, payload_raw, kind };
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
export class History {
    history = {};
    ttlSeconds;
    constructor(ttlSeconds) {
        this.ttlSeconds = Math.max(Math.round((ttlSeconds || 5)), 1);
        setInterval(this.rotate.bind(this), 1000);
    }
    exists(key) {
        let exists = false;
        for (let i = 0; i < this.ttlSeconds; i++) {
            let slice = this.history[`_${i}`];
            if (slice && slice[key]) {
                exists = true;
                break;
            }
        }
        if (!this.history["_0"]) {
            this.history["_0"] = {};
        }
        this.history["_0"][key] = true;
        return exists;
    }
    rotate() {
        for (let i = this.ttlSeconds; i > 0; i--) {
            let secKey = `_${i}`;
            if (this.history[secKey]) {
                if (i == this.ttlSeconds) {
                    delete this.history[secKey];
                }
                else {
                }
            }
        }
    }
}
//# sourceMappingURL=chain.js.map