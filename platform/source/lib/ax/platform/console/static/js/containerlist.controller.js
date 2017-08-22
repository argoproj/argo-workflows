var axconsoleApp = angular.module('axconsole', ['ui.bootstrap']);

axconsoleApp.config(function($interpolateProvider) {
    $interpolateProvider.startSymbol('[[');
    $interpolateProvider.endSymbol(']]');
});

axconsoleApp.controller('ContainerListController', function ($scope, $http, $location) {
    $scope.error = null;
    $scope.notify = null;

    $scope.pods = [];
    $scope.volumePools = [];

    // Detect if we are visited from context of k8 proxy url, for purposes linking to axworkflowadc
    if ($location.absUrl().search("/api/v1/proxy/namespaces") > 0) {
        $scope.adcBaseUrl = "../axworkflowadc"
    } else {
        $scope.adcBaseUrl = "http://axworkflowadc:8911"
    }
    
    $scope.monitorHostPrivateIp = null;
    $scope.monitorHostPublicDns = null;
    $scope.hosts = [];
    $scope.hostContainers = {};
    // Mapping of addresses to forwarded ports
    $scope.portForwards = {};
    // Cached list of exposed ports of containers from an inspect
    $scope.containerExposedPorts = {};
    // Determines if spinner should be displayed
    $scope.outstandingRequests = 0;
    // angular watch variable to indicate if container list needs to be redrawn
    $scope.redrawContainers = 0;
    // Terminated containers
    $scope.terminatedContainers = null;
    $scope.currentTab = "active";
    // Services
    $scope.services = {};
    $scope.jobPods = {};
    // Keep track of what user expanded in UI
    $scope.expandedSvs = {};

    $scope.errorCallback = function(response) {
        $scope.outstandingRequests--;
        if (response.data) {
            $scope.error = response.data;
        } else {
            $scope.error = "Request failed: " + response.config.method + " " + response.config.url;
        }
    }

    $scope.getPods = function() {
        $scope.outstandingRequests++;
        $http.get('v1/pods').then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.pods = response.data['data'];
                $scope.redrawContainers++;
            },
            $scope.errorCallback
        );
    }

    $scope.getVolumePools = function() {
        $scope.outstandingRequests++;
        $http.get('v1/volumepools').then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.volumePools = response.data['data'];
            },
            $scope.errorCallback
        );
    }

    $scope.deleteVolumePool = function(poolName) {
        $scope.outstandingRequests++;
        $http.delete('v1/volumepools/'+poolName).then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.notify = "Volume pool " + poolName + " successfully deleted.";
                $scope.getVolumePools();
            },
            $scope.errorCallback
        );
    }

    $scope.deleteVolume = function(poolName, volumeName) {
        $scope.outstandingRequests++;
        $http.delete('v1/volumepools/'+poolName+'/'+volumeName).then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.notify = "Volume " + volumeName + " in pool " + poolName + " successfully deleted.";
                $scope.getVolumePools();
            },
            $scope.errorCallback
        );
    }

    $scope.listTerminated = function() {
        $scope.outstandingRequests += 1;
        $http.get('api/terminated').then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.terminatedContainers = response.data.data;
            },
            $scope.errorCallback
        );
    }

    $scope.stopPod = function(pod, namespace) {
        $scope.outstandingRequests++;
        $http.delete('v1/pods/'+pod+'?namespace='+namespace).then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.notify = "Pod " + pod + " successfully stopped.";
                $scope.refreshActive();
            },
            $scope.errorCallback
        );
    }

    $scope.stopJob = function(serviceId) {
        $scope.outstandingRequests++;
        $http.delete('v1/jobs/'+serviceId).then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.notify = "Job " + serviceId + " requested sucessfully.";
                $scope.refreshJobs();
            },
            $scope.errorCallback
        );
    }


    $scope.listJobs = function() {
        $scope.outstandingRequests++;
        $http.get('v1/jobs').then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.services = response.data.data;
            },
            $scope.errorCallback
        );
    }

    $scope.listJobPods = function(serviceId) {
        $scope.outstandingRequests++;
        $http.get('v1/jobs/'+serviceId+'/pods').then(
            function successCallback(response) {
                $scope.outstandingRequests--;
                $scope.jobPods[serviceId] = response.data.data;
            },
            $scope.errorCallback
        );
    }

    $scope.refreshActive = function() {
        $scope.getPods();
    }
    
    $scope.refreshJobs = function() {
        $scope.listJobs();
        for (var svcId in $scope.expandedSvs) {
            if ($scope.expandedSvs[svcId]) {
                $scope.listJobPods(svcId);
            }
        }
    }

    $scope.refresh = function() {
        if ($scope.currentTab == "active") {
            $scope.refreshActive();
        } else if ($scope.currentTab == "terminated") {
            $scope.listTerminated();
        } else if ($scope.currentTab == "services") {
            $scope.refreshJobs();
        } else if ($scope.currentTab == "volumepools") {
            $scope.getVolumePools();
        }
    }

    $scope.setCurrentTab = function(tab) {
        $scope.currentTab = tab;
        $scope.refresh();
    }
    
    $scope.$watch(
        function(scope) { return scope.redrawContainers },
        function() {
            // HACK: figure out a proper way to force refresh of container list
            $scope.hostContainers = angular.copy($scope.hostContainers);            
        }
    );

    $scope.$watch(
        'hosts',
        function() {
            $scope.hostContainers = {};
            for (var i=0; i < $scope.hosts.length; i++) {
                $scope.outstandingRequests++;
                $http.get('v1/services/'+$scope.hosts[i]).then(
                    (function(h) {
                        return function(response) {
                            $scope.outstandingRequests--;
                            $scope.hostContainers[h] = response.data.data;
                        }
                     })($scope.hosts[i]),
                    $scope.errorCallback
                );
            }
        }
    );

    $scope.toggleExpand = function(serviceId) {
        $scope.expandedSvs[serviceId] = !$scope.expandedSvs[serviceId];
        if ($scope.expandedSvs[serviceId]) {
            $scope.listJobPods(serviceId);
        }
    }
    
    $scope.getSvcStatusColor = function(status) {
        if (status == 0) {
            return "success";
        }
        if (status == -1) {
            // Failed
            return "danger";
        }
        if (status == -2) {
            // Cancelled
            return "warning";
        }
        return "default";
    }

    $scope.getExitCodeColor = function(exitCode) {
        if (exitCode == 0) {
            return "success";
        }
        if (exitCode > 0 && exitCode < 128) {
            return "danger";
        }
        if (exitCode > 128) {
            // Typically indicates killed
            return "warning";
        }
        return "default";
    }

    $scope.dismissNotification = function() { $scope.notify = null; };
    $scope.dismissError = function() { $scope.error = null; };

    $scope.refreshActive();
});

axconsoleApp.directive('axContainerNames', function() {
    return {
        link: function(scope, element, attrs) {
            element.text(scope.ctr['Names'].join(", "));
        }
    };
});

//HumanDuration returns a human-readable approximation of a duration
//(eg. "About a minute", "4 hours ago", etc.).
function humanDuration(seconds) {
    seconds = Math.floor(seconds);
    if (seconds < 1) {
        return "Less than a second";
    } else if (seconds < 60) {
        return seconds + " seconds";
    }
    minutes = Math.floor(seconds / 60);
    if (minutes == 1) {
        return "About a minute";
    } else if (minutes < 60) {
        return minutes + " minutes";
    }
    hours = Math.floor(minutes / 60);
    if (hours == 1) {
        return "About an hour"
    } else if (hours < 48) {
        return hours + " hours";
    } else if (hours < 24*7*2) {
        return Math.floor(hours/24) + " days";
    } else if (hours < 24*30*3) {
        return Math.floor(hours/24/7) + " weeks";
    } else if (hours < 24*365*2) {
        return Math.floor(hours/24/30) + " months";
    }
    return Math.floor(hours/24/365) + " years";
}

axconsoleApp.directive('axDiedDate', function() {
    return {
        link: function(scope, element, attrs) {
            var date = new Date(scope.ctr.time*1000);
            var tzoffset = (new Date()).getTimezoneOffset() * 60000; //offset in milliseconds
            var localISOTime = (new Date(date - tzoffset)).toISOString().split('.')[0];
            var duration = (new Date()) - date;
            var durationStr = humanDuration(duration/1000);
            element.text(localISOTime + " (" + durationStr +" ago)");
        }
    };
});

