const socket = new WebSocket("ws://localhost:8080/ws");

socket.onopen = function (event) {
    console.log("WebSocket connection opened:", event);
};

socket.onmessage = function (event) {
    const logMessage = event.data;
    if (!logMessage) {
        console.error("Empty logMessage received");
        return;
    }

    console.log("WebSocket message received:", logMessage);
};

socket.onclose = function (event) {
    if (event.wasClean) {
        console.log("Closed cleanly, code=${event.code}, reason=${event.reason}");
    } else {
        console.error("Connection died");
    }
};

socket.onerror = function (error) {
    console.error("WebSocket Error:", error);
};
