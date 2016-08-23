
var myApp = angular.module('myApp', []);

myApp.factory('UploadService', ['$http',
    function ($http) {
    return {
    	uploadfile : function(files,success,error){
    		if (!files) {
    			console.log("Warning: no files to upload");
    			return;
    		}
    		console.log("Uploading files:",files);
			var url = '/upload';
			for ( var i = 0; i < files.length; i++) {
				var fd = new FormData();
				fd.append("select_files", files[i]);
				$http.post(url, fd, {
					withCredentials : false,
					headers : {
						'Content-Type' : undefined
					},
					transformRequest : angular.identity

				})
				.success(function(data) {
					console.log(data);
					success(data);
				})
				.error(function(data) {
					console.log(data);
					error(data);
				});
			}
		}       
    }
}]);

myApp.controller('MyController', ['$scope', '$location', '$window', '$http', '$timeout', 'UploadService',
    function($scope, $location, $window, $http, $timeout, UploadService) {

    	$scope.isFirstTime = true;

    	$scope.files = [];

    	$scope.uploadedFile = function(element) {
    		console.log('>>> uploadedFile');
			$scope.$apply(function($scope) {
				$scope.files = element.files;         
			});
			if ($scope.files.length>0) {
				$scope.addFile();
			}
			
		};

		$scope.isUploading = false;
		$scope.addFile = function() {
			console.log('>>> addFile');
			$scope.isUploading = true;
			$scope.isFirstTime = false;
			$scope.labelAnnotations = [];
			$scope.landmarkAnnotations = [];
			UploadService.uploadfile(
				$scope.files,
				function( msg ) { // success		
					console.log('uploaded',msg);
					if (msg && msg.responses && (msg.responses.length>0) && msg.responses[0].labelAnnotations) {
						$scope.labelAnnotations = msg.responses[0].labelAnnotations;
					} else {
						$scope.labelAnnotations = [];
					}
					for (var i = 0; i<$scope.labelAnnotations.length;i++) {
						$scope.labelAnnotations[i].score_p = Math.round(100*$scope.labelAnnotations[i].score);
					}

					if (msg && msg.responses && (msg.responses.length>0) && msg.responses[0].landmarkAnnotations) {
						$scope.landmarkAnnotations = msg.responses[0].landmarkAnnotations;
					} else {
						$scope.landmarkAnnotations = [];
					}
					for (var i = 0; i<$scope.landmarkAnnotations.length;i++) {
						$scope.landmarkAnnotations[i].score_p = Math.round(100*$scope.landmarkAnnotations[i].score);
					}
					$scope.isUploading = false;
				},
				function( msg ) { // error			
					console.log('error',msg);
					$scope.labelAnnotations = [];
					$scope.landmarkAnnotations = [];
					$scope.isUploading = false;
				}
			);
		};

    }
]);