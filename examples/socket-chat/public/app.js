(function () {

    const socket = chain.Socket('/socket')
    socket.connect()

    const channel = socket.channel("chat:lobby", {param1: 'foo'})
    channel.join()
        .on('ok', () => chain.log('Join', "success"))
        .on('error', err => chain.log('Join', "errored", err))
        .on('timeout', () => chain.log('Join', "timed out "))


    const $form = document.getElementById('chat-form')
    const $chatBox = document.getElementById('chat-box')
    const $inputName = document.getElementById('user-name')
    const $inputMessage = document.getElementById('user-msg')

    $form.addEventListener('submit', function (e) {
        e.preventDefault()

        if ($inputName.value === '') {
            return
        }

        channel.push('shout', {
            name: $inputName.value,
            body: $inputMessage.value
        })

        $inputMessage.value = ''
    })

    channel.on('shout', (message, ref, joinRef) => {
        let $p = document.createElement('p')
        $p.classList.add('c1')
        $p.insertAdjacentHTML('beforeend', `<b class="c2">${message.name}:</b> ${message.body}`)
        $chatBox.appendChild($p)
    })
})()
