
var dropbox, preview, message, url_stored = '/imsto/';
var roof = 'demo', userid = 0, token = '', default_roof = 'demo';
if (Dust) {Dust.dev = true;};

$(document).ready(function(){

	dropbox = $('#dropbox'), message = $('#message');

	function showMessage(msg){
		message.text(message.text() + msg + "\n\n").show();
	}

	dropbox.imgdrop({
		field_id: 'pic-input',
		field_name:'file',
		maxfiles: 9,
		maxfilesize: 1024,
		url: '/imsto/' + roof,
		// data: [{name: 'oper', value: 'add'}],
		error: function(err, file) {
			switch(err) {
				case 'BrowserNotSupported':
					Dust.alert('Your browser does not support HTML5 file uploads!');
					break;
				case 'TooManyFiles':
					Dust.alert('Too many files! Please select 9 at most! (configurable)');
					break;
				case 'FileTooLarge':
					Dust.alert(file.name+' is too large! Please upload files up to 2mb (configurable).');
					break;
				default:
					Dust.alert('Not select files to upload');
					break;
			}
		},
		dragEnter: function() {
			dropbox.attr('dragenter', true);
		},
		dragLeave: function() {
			//log('leave');
			dropbox.removeAttr('dragenter');
		},
		drop: function() {
			//log('drop');
			dropbox.removeAttr('dragenter');
		},
		imageCreated: function(img) {
			var li = $('<li />').addClass('new').append(img); // log(li);
			li.append('<div class="left"><label class="lbl"></label><input type="text" name="tags" value="" placeholder="tags with comma separated"/></div>');
			li.append($('<span class="btn" />').addClass('btn ui-corner-all ui-icon ui-icon-trash').attr('title','delete').click(function(){$(this).parent().fadeOut(function(){$(this).remove();});return false;}));
			$("#image-pic-list").prepend(li);
		},
		imageLoaded: function(img) {
			$(img).parent().find('.lbl').text($(img).attr('title'));
		}
	});

	$("#progressbar").progressbar().hide();

	// $("#image-pic-list").sortable({
	// 	//placeholder: 'ui-state-highlight',
	// 	update: function(event, ui) {
	// 		//console.log(event, ui);
	// 	}
	// });
	// $("#image-pic-list").disableSelection();
	$("#upload_form").submit(function(form){
		if ($("#image-pic-list li").length == 0) return false;
		$("#image-pic-list li.new").each(function(i){
			$(this).attr('id','li_new'+i);
		});
		var params = [{name:'image_id', value:''},{name: 'act', value: 'image_store'}];//$(form).serializeArray();
		// var sorted = $("#image-pic-list").sortable( 'toArray' );
		// $.each(sorted, function(i, item){
		// 	var res = item.match(/(.+)[-=_](.+)/);
		// 	if(res) params.push({name:'li[]',value:res[2]});
		// });
		var files = [], imgs = [];
		$("li.new").each(function(i, li){
			var img = $("img", this).get(0);
			if (typeof img.file !== "undefined") {
				var tags = $('input[name=tags]', this);
				if (tags.length > 0) {img.file.tags = tags.val();}
				files.push(img.file);
				imgs.push(img);
			}
		});
		$("li input[type=text].changed").each(function(i,input){
			params.push({name:input.name, value: input.value});
		});
		log(params);
		//alert('params: ' + dump(params));
		//alert('files: ' + dump(files));
		//$("#indicator").text('start process');
		log('upload start ', files.length);
		if(files.length > 0) $("#progressbar").show();
		if (files.length === 0) {
			Dust.alert('New file not found, please select one or more files');
			return false;
		}

		$.imgupload({

			url: '/imsto/' + roof,
			files: files,
			images: imgs,
			data: [
					{name: 'roof', value: roof}
					,{name: 'api_key', value: api_key}
					,{name: 'user', value: userid}
					,{name: 'token', value: token}
			],
			uploadStarted: function(i, file, len){
				// a file began uploading
				// i = index => 0, 1, 2, 3, 4 etc
				// file is the actual file of the index
				// len = total files user dropped
				log('start ' + i + ': ' + file.name)
			},
			uploadFinished: function(i, file, res, time) {
				// response is the data you got back from server in JSON format.
				log('finished: ' + i + ': ' + file.name + ', ' + time + ' server responsed', res);
				if (res.meta.ok === true) {
					// alertAjaxResult(res);
					if ($.isArray(res.data)) {
						$.each(res.data, function(i, item){
							if (typeof item.error == "string") {
								showMessage('save ' + (i+1) + ': ' + file.name + ' failed, error: ' + item.error);
							}
							else {
								$(imgs[i]).parent().removeClass('new');
								showMessage('save ' + (i+1) + ': ' + file.name + ' OK, path: ' + item.path);
							}
						});
					}
					return true;
				}
				if (!!res.errors) {
					showMessage('' + (i+1) + 'save ' + file.name + ' failed, error: ' + res.errors.message + "\ncode: " + res.errors.code);
					return false;
				}

			},
			progressUpdated: function(i, file, progress) {
				// this function is used for large files and updates intermittently
				// progress is the integer value of file being uploaded percentage to completion
				log('progress: ' + i + ': ' + progress);
				$("#progressbar").progressbar("option", "value", progress);
			},
			speedUpdated: function(i, file, speed) {
				// speed in kb/s
				log('speed: ' + i + ': ' + speed);
			},
			beforeEach: function(file) {
				// file is a file object
				// return false to cancel upload

				if(!file.type.match(/^image\//)){
					Dust.alert('Only images are allowed!');

					// Returning false will cause the
					// file to be rejected
					return false;
				}
			},
			afterAll: function() {
				// runs after all files have been uploaded or otherwise dealt with
				log('all done!');
				/*Dust.alert('all done!', function(){
					// TODO: refresh page?
					setTimeout(function(){
						message.fadeOut(900).text('').hide();
					}, 6000);

				});*/
				$("#pic-input").val(null);
			}
		});

		return false;
	});

	$(".button").button();

	$("#clear").click(function(){
		$("#pic-input").val(null);
	});

	$("input:submit").hide();

	function req_token () {
		// roof = $("input:radio[name=roof]:checked").val();
		// console.log(roof)
		$.post('/imsto/token', {
			roof: roof,
			api_key: api_key,
			user: userid
		}, function(res){
			if (res.meta.ok === true) {
				token = res.meta.token;
				// log(token)
			} else if (typeof res.error == "string") {
				showMessage(res.error)
			}
			$("input:submit").show();
		}, 'json');
	}

	var now = new Date();

	$.getJSON('/imsto/roofs', function(res){
		// console.log(typeof res.roofs)
		var roofs = {};
		res.data.forEach(function(name, idx){roofs[name]=name.toUpperCase()});
		$('#radios1').radioButtons({
			data: roofs,
			name: 'roof',
			selected: default_roof
		});

		req_token();

		$("input[name=roof]:radio").click(function(){
			// log(this.value);
			if (this.checked) {
				roof = this.value
				req_token();
			}
		});
	});

	log(now);
});
