var axconsoleApp = angular.module('axconsole', []);

axconsoleApp.config(function($interpolateProvider) {
    $interpolateProvider.startSymbol('[[');
    $interpolateProvider.endSymbol(']]');
});

axconsoleApp.controller('ExecController', function ($scope, $http) {
    $scope.addr = "";
    $scope.command = "sh";
    $scope.connected = false;
    $scope.connect = connect;
    $scope.disconnect = disconnect;
    $scope.error = "";
    var url = window.location.href;
    var urlParts = window.location.href.split("/");
    var appurl = urlParts.slice(0, urlParts.length-3).join("/");

    var parser = document.createElement('a');
    parser.href = window.location.href;

    var pathnameParts = parser.pathname.split('/');
    $scope.service_id = pathnameParts[pathnameParts.length-2];

    var term;
    var websocket;

    function connect() {
        var termWidth = Math.round(window.innerWidth / 7);
        var termHeight = Math.round(window.innerHeight / 20);
        $scope.addr = appurl + "/v1/" + pathnameParts.slice(pathnameParts.length-3, pathnameParts.length-1).join("/") + "/exec" + window.location.search
        if (parser.protocol === "https:") {
            $scope.addr = $scope.addr.replace('https://', 'wss://')
        } else {
            $scope.addr = $scope.addr.replace('http://', 'ws://')
        }
        websocket = new WebSocket($scope.addr);
        websocket.binaryType = 'arraybuffer';
        websocket.onopen = function(evt) {
            $scope.error = "";
            term = new Terminal({
                cols: termWidth,
                rows: termHeight,
                screenKeys: true,
                useStyle: true,
                cursorBlink: true,
            });
            term.on('data', function(data) {
                websocket.send(data);
            });
            term.on('title', function(title) {
                document.title = title;
            });
            term.open(document.getElementById('container-terminal'));
            websocket.onmessage = function(evt) {
                if (event.data instanceof ArrayBuffer) {
                    var bytearray = new Uint8Array(evt.data);
                    var result = "";
                    for (var i=0; i < bytearray.length; i++) {
                        result += String.fromCharCode(bytearray[i]);
                    }
                    term.write(result);
                } else {
                    term.write(evt.data);
                }
            }
            websocket.onclose = function(evt) {
                if (evt.wasClean == true) {
                    term.write("Session terminated");
                } else {
                    $scope.error = "Unclean socket close";
                    $scope.$applyAsync();
                }
                term.destroy();
            }
            websocket.onerror = function(evt) {
                if (typeof console.log == "function") {
                    //console.log(evt)
                }
                $scope.error = evt;
                $scope.$applyAsync();
            }
        }
        $scope.connected = true;
    }

    function disconnect() {
        $scope.connected = false;
        if (websocket != null) {
            websocket.close();
        }

        if (term != null) {
            term.destroy();
        }
    }

});