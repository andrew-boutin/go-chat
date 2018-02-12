var username = document.getElementById("username");
var input = document.getElementById("input");
var textArea = document.getElementById("textarea");

var ws = new WebSocket('ws://' + window.location.host + '/ws');

ws.onopen = function(){
    displayMsg("Connected to server.");
};

ws.onerror = function(){
    displayMsg("Error communicating with server.");
};

ws.onclose = function(){
    displayMsg("Connection to server closed.");
};

ws.onmessage = function(msgevent){
    var obj = JSON.parse(msgevent.data);
    displayMsg(obj.username + ": " + obj.message);
};

function displayMsg(msg){
    textArea.value += "\n" + msg;
};

function sendMsg() {
    ws.send('{"username":"' + username.value + '","message":"' + input.value + '"}');
};