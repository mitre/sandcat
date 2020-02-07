function displayCommand(){
    $('#delivery-command').text(atob(document.getElementById("dcommands").value));
}
let copyCommandBtn = document.querySelector('#copyCommand');
copyCommandBtn.addEventListener('click', function(event) {
    let command = document.querySelector('#delivery-command');
    let range = document.createRange();
    range.selectNode(command);
    window.getSelection().addRange(range);
    try {
        document.execCommand('copy');
    } catch(err) {
        console.log('Oops, unable to copy');
    }
    window.getSelection().removeAllRanges();
});