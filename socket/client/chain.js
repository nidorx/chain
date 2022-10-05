(function () {

    const SOCKET = 'Socket';
    const CHANNEL = 'Channel';
    const TRANSPORT = 'Transport';

    const MESSAGE_KIND_PUSH = 0;
    const MESSAGE_KIND_REPLY = 1;
    const MESSAGE_KIND_BROADCAST = 2;

    let logGroupLen = Math.max(TRANSPORT.length, CHANNEL.length, SOCKET.length);
    const chain = window.chain = {
        Socket: Socket,
        Transport: {SSE: TransportSSE},
        Retry: Retry,
        Events: Events,
        Push: Push,
        Channel: Channel,
        Encode: encode,
        Decode: decode,
        Debug: true,
        log: (group, template, ...params) => {
            if (typeof group === 'string' && chain[`Debug${group}`] === false) {
                return
            }
            if (chain.Debug || (typeof group === 'string' && chain[`Debug${group}`])) {
                if (typeof template !== 'string') {
                    params = [template, ...params]
                    template = ''

                    if (typeof group !== 'string') {
                        params = [group, ...params]
                        group = ''
                    }
                }

                logGroupLen = Math.max(logGroupLen, group.length)
                console.log(
                    `${`                           ${group}`.substr(-logGroupLen)}: ${template}`,
                    ...params
                )
            }
        }
    }

    function encode(message) {
        let {joinRef, ref, topic, event, payload} = message
        let s = JSON.stringify([MESSAGE_KIND_PUSH, joinRef, ref, topic, event, payload]);
        return s.substr(1, s.length - 2)
    }

    function decode(rawMessage) {
        // Push      = [kind, joinRef, ref,  topic, event, payload]
        // Reply     = [kind, joinRef, ref, status,        payload]
        // Broadcast = [kind,                topic, event, payload]
        let [kind, joinRef, ref, topic, event, payload] = JSON.parse(`[${rawMessage}]`)
        if (kind === MESSAGE_KIND_REPLY) {
            payload = {status: topic === 0 ? 'ok' : 'error', response: event};
            event = 'stx_reply';
            topic = undefined
        } else if (kind === MESSAGE_KIND_BROADCAST) {
            payload = topic;
            event = ref;
            topic = joinRef;
            joinRef = ref = undefined;
        }
        return {joinRef: joinRef, ref: ref, topic: topic, event: event, payload: payload, kind: kind}
    }

    /**
     *
     * @param endpoint
     * @param options
     * @return {any}
     * @constructor
     */
    function Socket(endpoint, options = {}) {

        const events = Events()
        const channels = [];
        const sendBuffer = []

        let ref = 1;
        let connected = false;

        let transport = options.transport || TransportSSE;
        let conn = transport(endpoint, options.transportOptions || {})
        conn.on("open", onConnOpen)
        conn.on("error", onConnError)
        conn.on("message", onConnMessage)
        conn.on("close", onConnClose)

        const socket = {
            timeout: options.timeout || 30000,
            rejoinInterval: options.rejoinInterval || [1000, 2000, 5000, 10000],
            reconnectInterval: options.reconnectInterval || [10, 50, 100, 150, 200, 250, 500, 1000, 2000, 5000],
            on: events.on.bind(events),
            isConnected: () => connected,
            push: push,
            channel: channel,
            ref: makeRef,
            remove: remove,
            disconnect: disconnect,
            leaveOpenTopic: leaveOpenTopic,
            connect: conn.connect
        }

        return socket

        /**
         * Initiates a new channel for the given topic
         *
         * @param {string} topic
         * @param {Object} params - Parameters for the channel
         * @returns {Object}
         */
        function channel(topic, params = {}, options = {}) {
            let channel = Channel(topic, params, socket, options)
            channels.push(channel)
            return channel
        }

        function remove(channel) {
            let idx = channels.indexOf(channel)
            if (idx >= 0) {
                channels.splice(idx, 1)
            }
        }

        function leaveOpenTopic(topic) {
            let dupChannel = channels.find(c => {
                return c.topic === topic && (c.isJoined() || c.isJoining())
            })
            if (dupChannel) {
                chain.log(SOCKET, 'leaving duplicate topic "%s"', topic)
                dupChannel.leave()
            }
        }

        function push(message) {
            let {topic, event, payload, ref, joinRef} = message
            const data = encode(message)
            if (connected) {
                chain.log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload)
                conn.send(data)
            } else {
                chain.log(SOCKET, 'push %s %s (%s, %s) [scheduled]', topic, event, joinRef, ref, payload)
                sendBuffer.push(() => {
                    chain.log(SOCKET, 'push %s %s (%s, %s)', topic, event, joinRef, ref, payload)
                    conn.send(data)
                })
            }
        }

        /**
         * Return the next message ref, accounting for overflows
         *
         * @returns {number}
         */
        function makeRef() {
            ref++;
            if (ref === Number.MAX_SAFE_INTEGER) {
                ref = 1
            }

            return ref
        }

        function onConnOpen() {
            chain.log(SOCKET, 'connected to %s', endpoint)

            connected = true

            if (sendBuffer.length > 0) {
                // flush send buffer
                sendBuffer.forEach(callback => callback())
                sendBuffer.splice(0)
            }

            events.emit('open')
        }

        function onConnClose(event) {
            connected = false
            events.emit('close')
        }

        function onConnMessage(data) {
            let message = decode(data)
            let {topic, event, payload, ref, joinRef} = message
            chain.log(SOCKET, 'receive %s %s %s',
                topic || '', event || '', (ref || joinRef) ? (`(${joinRef || ''}, ${ref || ''})`) : '', payload
            )

            channels.forEach(channel => {
                channel.trigger(event, payload, topic, ref, joinRef)
            })

            events.emit('message', message)
        }

        function onConnError(error) {
            events.emit('error', error)
        }

        function disconnect(callback, code, reason) {
            conn.close()
        }
    }

    const CHANNEL_STATE_CLOSED = 0;
    const CHANNEL_STATE_ERRORED = 1;
    const CHANNEL_STATE_JOINED = 2;
    const CHANNEL_STATE_JOINING = 3;
    const CHANNEL_STATE_LEAVING = 4;

    /**
     *
     * @param topic
     * @param chanParams
     * @param socket
     * @param options
     * @constructor
     */
    function Channel(topic, chanParams, socket, options) {
        const events = Events();
        const pushBuffer = [];

        let state = CHANNEL_STATE_CLOSED;
        let joinedOnce = false;
        let timeout = socket.timeout;

        const channel = {
            join: join,
            leave: leave,
            push: push,
            trigger: trigger,
            on: events.on.bind(events),
            onClose: events.on.bind(events, "stx_close"),
            onError: events.on.bind(events, "stx_error"),
            isClosed: stateIsFn(CHANNEL_STATE_CLOSED),
            isErrored: stateIsFn(CHANNEL_STATE_ERRORED),
            isJoined: stateIsFn(CHANNEL_STATE_JOINED),
            isJoining: stateIsFn(CHANNEL_STATE_JOINING),
            isLeaving: stateIsFn(CHANNEL_STATE_LEAVING),
            topic: () => topic,
            joinRef: joinRef,
        }

        let rejoinRetry = Retry(() => {
            if (socket.isConnected()) {
                rejoin()
            }
        }, socket.rejoinInterval)

        let cancelOnSocketError = socket.on('error', () => {
            rejoinRetry.reset()
        })

        let cancelOnSocketOpen = socket.on('open', () => {
            rejoinRetry.reset()
            if (channel.isErrored()) {
                rejoin()
            }
        })

        let joinPush = Push(socket, channel, 'stx_join', chanParams, timeout)
            .on('ok', () => {
                state = CHANNEL_STATE_JOINED
                rejoinRetry.reset()
                pushBuffer.forEach(push => push.send())
                pushBuffer.splice(0)
            })
            .on('error', () => {
                state = CHANNEL_STATE_ERRORED
                if (socket.isConnected()) {
                    rejoinRetry.retry()
                }
            })
            .on('timeout', () => {
                chain.log(CHANNEL, 'timeout %s (%s)', topic, joinRef(), joinPush.timeout())

                // leave (if joined on server)
                Push(socket, channel, 'stx_leave', {}, timeout).send()

                state = CHANNEL_STATE_ERRORED
                joinPush.reset()
                if (socket.isConnected()) {
                    rejoinRetry.retry()
                }
            })


        channel.onClose(() => {
            if (channel.isClosed()) {
                return
            }
            chain.log(CHANNEL, 'close %s %s', topic, joinRef())

            cancelOnSocketOpen();
            cancelOnSocketError();
            rejoinRetry.reset();
            state = CHANNEL_STATE_CLOSED;
            socket.remove(channel);
        })

        channel.onError(reason => {
            chain.log(CHANNEL, 'error %s', topic, reason)

            if (channel.isJoining()) {
                joinPush.reset()
            }
            state = CHANNEL_STATE_ERRORED
            if (socket.isConnected()) {
                rejoinRetry.retry()
            }
        })

        channel.on('stx_reply', (payload, ref) => {
            trigger(`chan_reply_${ref}`, payload)
        })

        // Overridable message hook
        // Receives all events for specialized message handling before dispatching to the channel callbacks.
        // Must return the payload, modified or unmodified
        let onMessage = typeof options.OnMessage == 'function' ? options.OnMessage : (_e, payload) => payload

        return channel

        function joinRef() {
            return joinPush.ref()
        }

        /**
         * Join the channel
         *
         * @param {number} p_timeout
         * @return {*}
         */
        function join(p_timeout = timeout) {
            if (joinedOnce) {
                throw new Error("tried to join multiple times. 'join' can only be called a single time per channel instance")
            } else {
                timeout = p_timeout
                joinedOnce = true
                rejoin()
                return joinPush
            }
        }

        function rejoin() {
            if (channel.isLeaving()) {
                return
            }
            socket.leaveOpenTopic(topic)
            state = CHANNEL_STATE_JOINING
            joinPush.resend(timeout)
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
         * @param {number} p_timeout
         * @returns {Object}
         */
        function leave(p_timeout = timeout) {
            rejoinRetry.reset();
            joinPush.cancelTimeout();

            state = CHANNEL_STATE_LEAVING;

            let onClose = () => {
                chain.log(CHANNEL, 'leave %s', topic)
                trigger('stx_close', 'leave')
            }

            let leavePush = Push(socket, channel, 'phx_leave', {}, p_timeout)
                .on('ok', onClose)
                .on('timeout', onClose);

            leavePush.send();

            if (!canPush()) {
                leavePush.trigger('ok', {})
            }

            return leavePush
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
         * @return {any}
         */
        function push(event, payload, p_timeout = timeout) {
            payload = payload || {}
            if (!joinedOnce) {
                throw new Error(`tried to push '${event}' to '${topic}' before joining. Use channel.join() before pushing events`)
            }

            let push = Push(socket, channel, event, payload, p_timeout)
            if (canPush()) {
                push.send()
            } else {
                push.startTimeout()
                pushBuffer.push(push)
            }
            return push
        }

        function trigger(event, payload, p_topic, ref, p_joinRef) {
            if (p_topic && topic !== p_topic) {
                // to other channel
                return
            }
            if (p_joinRef && p_joinRef !== joinRef()) {
                // outdated message or to other channel
                return
            }

            let handledPayload = onMessage(event, payload, ref, p_joinRef);
            if (payload && !handledPayload) {
                throw new Error("channel onMessage callbacks must return the payload, modified or unmodified")
            }

            events.emit(event, handledPayload, ref, p_joinRef || joinRef())
        }

        function canPush() {
            return socket.isConnected() && channel.isJoined()
        }

        function stateIsFn(s) {
            return () => state === s
        }
    }


    /**
     * Create a Push event
     *
     * @param socket
     * @param channel
     * @param event
     * @param payload
     * @param timeout
     * @return {any}
     * @constructor
     */
    function Push(socket, channel, event, payload, timeout) {
        const events = Events()

        payload = payload || {}
        let received = null
        let timer = null
        let sent = false
        let ref = socket.ref()
        let refEvent
        let refEventCancel

        const push = {
            on: (event, callback) => {
                if (hasReceived(event)) {
                    queueMicrotask(callback.bind(null, received.response))
                } else {
                    events.on(event, callback)
                }
                return push
            },
            send: send,
            resend: resend,
            reset: reset,
            trigger: trigger,
            timeout: () => timeout,
            cancelTimeout: cancelTimeout,
            startTimeout: startTimeout,
            ref: () => ref
        }
        return push

        function send() {
            if (hasReceived('timeout')) {
                return
            }
            startTimeout()
            sent = true
            socket.push({
                topic: channel.topic(),
                event: event,
                payload: payload,
                ref: ref,
                joinRef: channel.joinRef()
            })
        }

        function resend(p_timeout) {
            timeout = p_timeout
            reset()
            send()
        }

        function reset() {
            if (refEventCancel) {
                refEventCancel()
                refEventCancel = null
            }
            sent = false
            ref = null
            refEvent = null
            received = null
        }

        function cancelTimeout() {
            clearTimeout(timer)
            timer = null
        }

        function startTimeout() {
            if (timer) {
                cancelTimeout()
            }
            ref = socket.ref()
            refEvent = `chan_reply_${ref}`

            refEventCancel = channel.on(refEvent, payload => {
                if (refEventCancel) {
                    refEventCancel()
                    refEventCancel = null
                }
                cancelTimeout()
                received = payload
                let {status, response, _ref} = payload
                events.emit(status, response)
            })

            timer = setTimeout(() => {
                trigger("timeout", {})
            }, timeout)
        }

        function hasReceived(status) {
            return received && received.status === status
        }

        function trigger(status, response) {
            // event, payload, topic, ref, joinRef
            channel.trigger(refEvent, {status, response})
        }
    }

    /**
     * Channel transport using server-sent events
     *
     * @param endpoint
     * @param options
     * @return {{send: send, close: close, on: any}}
     * @constructor
     */
    function TransportSSE(endpoint, options = {}) {
        const events = Events();

        let source;
        let pushEndpoint = (endpoint.split('?')[0] + '/sse').replaceAll(/[/]+/g, '/');

        function send(data) {
            // fire and forget
            fetch(pushEndpoint, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: data,
            }).catch((error) => {
                chain.error(TRANSPORT, 'send error', error, data);
            });
        }

        function connect() {
            source = new EventSource(parseUrl(endpoint, '/sse', options.params));

            source.onmessage = (event) => {
                chain.log(TRANSPORT, 'message', event)
                events.emit('message', event.data)
            }

            source.onerror = (event) => {
                chain.log(TRANSPORT, 'error', event)
                events.emit('error')
            }

            source.onopen = (event) => {
                chain.log(TRANSPORT, 'open', event)
                events.emit('open')
            }
        }

        function close() {
            if (!source) {
                return
            }
            chain.log(TRANSPORT, 'close')
            source.close()
            events.emit('close')
        }

        return {
            on: events.on.bind(events),
            send: send,
            connect: connect,
            close: close,
        }
    }

    /**
     * Timer to retry callback
     *
     * @param callback
     * @param intervals
     * @return {{reset: reset, retry: retry}}
     * @constructor
     */
    function Retry(callback, intervals) {
        intervals = intervals.slice(0).sort()
        let maxInterval = Math.max(...intervals)
        let timer = null;
        let tries = 0;

        return {
            reset: reset,
            retry: retry
        }

        function reset() {
            tries = 0;
            clearTimeout(timer);
        }

        function retry() {
            clearTimeout(timer);
            timer = setTimeout(() => {
                tries++;
                callback();
            }, intervals[tries] || maxInterval);
        }
    }

    /**
     * Copyright 2016 Andrey Sitnik <andrey@sitnik.ru>, https://github.com/ai/nanoevents/blob/main/LICENSE
     *
     * @return {{emit(*, ...[*]): void, on(*, *): function(): void}|(function(): void)|*}
     * @constructor
     */
    function Events() {
        const events = {};
        return {
            emit(event, ...args) {
                let callbacks = events[event] || []
                for (let i = 0, length = callbacks.length; i < length; i++) {
                    callbacks[i](...args)
                }
            },
            on(event, cb) {
                events[event]?.push(cb) || (events[event] = [cb])
                // off
                return () => {
                    let callbacks = events[event]
                    if (callbacks) {
                        let idx = callbacks.indexOf(cb);
                        if (idx >= 0) {
                            callbacks.splice(idx, 1)
                        }
                    }
                }
            }
        }
    }

    function parseUrl(endpoint, suffix, params) {
        let parts = endpoint.split('?');
        let basePath = (parts[0] + suffix).replaceAll(/[/]+/g, '/');
        let queryString = parts[1] || '';
        if (params) {
            for (let key in params) {
                if (queryString.length > 0) {
                    queryString = queryString + '&'
                }
                queryString = queryString + `${key}=${encodeURIComponent(params[key])}`
            }
        }
        return `${basePath}?${queryString}`;
    }

    return chain
})()
