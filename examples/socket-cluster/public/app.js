import * as chain from '/chain.js';

let priority = 8081;
let servers = [8081];

const socket = new chain.Socket({
    transport: [
        {
            name: "SSE",
            cors: true,
        }
    ],
    getNodesInterval: 5,
    getNodes: async () => {
        return fetch('/node')
            .then(res => res.json())
            .then((res) => {
                let host = window.location.host.split(':')[0];

                servers = res || [];

                if (!servers.includes(priority)) {
                    priority = servers[0];
                }

                servers.sort((a, b) => {
                    if (a == priority) {
                        return -1;
                    }
                    if (b == priority) {
                        return 1;
                    }
                    return a - b;
                });

                setTimeout(updateServerListButton, 50);

                return servers.map((port) => {
                    return `http://${host}:${port}/socket`
                });
            });
    }
})
socket.connect()

socket.channel("chat:lobby", { param1: 'foo' })
    .on('ok', () => chain.log('Join', "success"))
    .on('error', err => chain.log('Join', "errored", err))
    .on('timeout', () => chain.log('Join', "timed out "))
    .on('message', onMessage)
    .join();

document.getElementById('node-add').onclick = () => {
    fetch('/node', { method: 'post' }).then(res => { });
};

document.getElementById('node-del').onclick = () => {
    fetch('/node', { method: 'delete' }).then(res => { });
};

const chatBox = document.getElementById('chat-box');

function onMessage(message, ref, joinRef) {
    let div = document.createElement('div')
    div.classList.add('c1');
    div.innerHTML = JSON.stringify(message);
    chatBox.insertBefore(div, chatBox.firstChild);

    if (chatBox.childNodes.length > 50) {
        chatBox.removeChild(chatBox.lastChild);
    }
}

function updateServerListButton() {
    let container = document.getElementById('server-list');
    container.innerHTML = '';

    servers.forEach((port) => {
        let button = document.createElement('button')
        button.innerHTML = `:${port}`;
        button.onclick = (e) => {
            e.preventDefault();
            e.stopPropagation();
            priority = port;
            updateServerListButton();
        };

        if (priority == port) {
            button.classList.add('preferred');
        }

        container.appendChild(button);
    })
}
