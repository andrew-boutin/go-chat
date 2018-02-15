var input = document.getElementById("input");
var textArea = document.getElementById("textarea");

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
    displayMsg(obj.username + ": " + obj.message);
};

function displayMsg(msg){
    textArea.innerHTML += "<br>" + msg;
};

function sendMsg() {
    ws.send('{"message":"' + input.value + '"}');
};