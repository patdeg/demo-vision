
var myApp = angular.module('myApp', []);


// Upload Service (factory) to load file (image) to backend
myApp.factory('UploadService', ['$http',
    function ($http) {
    return {
    	uploadfile : function(files,success,error){
    		if (!files) {
    			console.log("Warning: no files to upload");
    			return;
    		}    	
    		// Upload destination	
			var url = '/upload';

			// Loop through files and upload one by one 
			// Overcomplification in the simple example 
			// with unique file, but usefull down the road
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
					success(data);
				})
				.error(function(data) {					
					error(data);
				});
			}
		}       
    }
}]);

myApp.controller('MyController', ['$scope', '$location', '$window', '$http', '$timeout', 'UploadService',
    function($scope, $location, $window, $http, $timeout, UploadService) {

    	// Boolean to change button label from "Select New Picture" to "Select Picture"
    	$scope.isFirstTime = true;

    	// Boolean to change text in the page when uploading
		$scope.isUploading = false;

    	// List of files to upload
    	$scope.files = [];

    	// Annotation arrays
    	$scope.labelAnnotations = [];
		$scope.landmarkAnnotations = [];
		$scope.logoAnnotations = [];
		$scope.textAnnotations = "";

    	// Function called when the list of file(s) in the form change
    	$scope.uploadedFile = function(element) {    		
			$scope.$apply(function($scope) {
				$scope.files = element.files;         
			});
			// When adding/changing a file, automatically upload the file to the server
			if ($scope.files.length>0) {
				$scope.addFile();
			}
			
		};

		// Function to load file to the server
		$scope.addFile = function() {

			// Set Upload mode
			$scope.isUploading = true;
			$scope.isFirstTime = false;
			$scope.labelAnnotations = [];
			$scope.landmarkAnnotations = [];
			$scope.logoAnnotations = [];
			$scope.textAnnotations = "";

			UploadService.uploadfile(
				$scope.files,
				function( msg ) { // success	
					console.log("Success:",msg);						
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

					if (msg && msg.responses && (msg.responses.length>0) && msg.responses[0].logoAnnotations) {
						$scope.logoAnnotations = msg.responses[0].logoAnnotations;
					} else {
						$scope.logoAnnotations = [];
					}
					for (var i = 0; i<$scope.logoAnnotations.length;i++) {
						$scope.logoAnnotations[i].score_p = Math.round(100*$scope.logoAnnotations[i].score);
					}

					if (msg && msg.responses && (msg.responses.length>0) && msg.responses[0].textAnnotations && (msg.responses[0].textAnnotations.length>0)) {
						$scope.textAnnotations = msg.responses[0].textAnnotations[0].description;
					} else {
						$scope.textAnnotations = "";
					}					

					$scope.isUploading = false;
				},
				function( msg ) { // error			
					console.log('Error:',msg);
					$scope.labelAnnotations = [];
					$scope.landmarkAnnotations = [];
					$scope.logoAnnotations = [];
					$scope.textAnnotations = "";
					$scope.isUploading = false;
				}
			);
		};
    }
]);