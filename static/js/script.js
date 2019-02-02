var input = document.getElementById("input");
var button = document.getElementById("button");
var textArea = document.getElementById("textarea");
var users = document.getElementById("users");
var username = "";

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
    else if(obj.username == "server" && obj.message.startsWith("Hello ")) {
        username = obj.message.substring(obj.message.indexOf("Hello ") + "Hello ".length);
    }
    else { // Add a chat message
        var fromSelf = obj.username == username;
        displayMsg(obj.username + ": " + obj.message, fromSelf);
    }
};

function displayMsg(msg, fromSelf=false){
    textArea.innerHTML += "<br>"
    var newMsg = msg;
    if(fromSelf) {
        newMsg = "<i>" + newMsg + "</i>";
    }
    textArea.innerHTML += newMsg;
};

function sendMsg() {
    ws.send('{"username":"' + username + '", "message":"' + input.value + '"}');
};
