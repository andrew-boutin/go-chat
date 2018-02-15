var input = document.getElementById("input");
var button = document.getElementById("button");
var textArea = document.getElementById("textarea");
var users = document.getElementById("users");

input.addEventListener("keyup", function(event) {
    event.preventDefault();
    if(event.keyCode == 13) {
        button.click();
    }
});

button.addEventListener("click", function(event) {
    input.value = "";
});

var ws = new WebSocket('ws://' + window.location.host + '/ws');

ws.onopen = function(){
    displayMsg("client: Connected to server.");
};

ws.onerror = function(){
    displayMsg("client: Error communicating with server.");
};

ws.onclose = function(){
    displayMsg("client: Connection to server closed.");
};

ws.onmessage = function(msgevent){
    var obj = JSON.parse(msgevent.data);

    // Update the users information
    if(obj.username == "users") {
        users.innerHTML = "Users: " + obj.message;
    }
    else { // Add a chat message
        displayMsg(obj.username + ": " + obj.message);
    }
};

function displayMsg(msg){
    textArea.innerHTML += "<br>" + msg;
};

function sendMsg() {
    ws.send('{"message":"' + input.value + '"}');
};