var destroyTimer = null;

$(function() {
	RT.plugins.notify = function(userOpts) {

		var opts = $.extend({
			theme : "",
			duration : 2000,
			position : "bottomright",
			animate : "fade",
			title : "",
			msg : "",
			destroyNotify : function() {
				$("#notifyBox").stop().fadeOut();
				$("#notifyBox").css("opacity",1);
				clearTimeout(destroyTimer);
			}
		},userOpts);
		
		clearTimeout(destroyTimer);
			
		if(!$("#notifyBox").length) {
			$("body").append("\
				<div id='notifyBox'>\
					<div class='notifyTitle'></div>\
					<div class='notifyBody'></div>\
				</div>\
			");
		}
		
		// if any are visible
		if($("#notifyBox:visible").length) {
			opts.destroyNotify();
		}
			
		// clear any classes
		$("#notifyBox").removeAttr("class");
		
		$("#notifyBox").hover(function() {
			clearTimeout(opts.destroyTimer);
		}, function() {
			opts.destroyTimer = setTimeout(opts.destroyNotify,opts.duration);
		});
			
		// put the text in
		$("#notifyBox .notifyTitle").html(opts.title);
		$("#notifyBox .notifyBody").html(opts.msg);
		
		// trick for icons
		if(!opts.title) {
			$("#notifyBox .notifyTitle").css("float","left");
		} else {
			$("#notifyBox .notifyTitle").css("float","none");
		}	
			
		// theme
		$("#notifyBox").addClass(opts.theme);	
			
		// place it
		switch(opts.position) {
			case "bottomright":
				$("#notifyBox").css({
					"bottom":0,
					"right":0
				});
				break;
			case "bottomleft":
				$("#notifyBox").css({
					"bottom":0,
					"left":0
				});
				break;
			case "topright":
				$("#notifyBox").css({
					"top":0,
					"right":0
				});
				break;
			case "topleft":
				$("#notifyBox").css({
					"top":0,
					"left":0
				});
				break;
			case "top":
				$("#notifyBox").css({
					"top":0,
					"left":0,
					"max-width":"none",
					"width":"100%",
					"margin":"0px"
				}).addClass("full");
				break;	
			case "bottom":
				$("#notifyBox").css({
					"bottom":0,
					"left":0,
					"max-width":"none",
					"width":"100%",
					"margin":"0px"
				}).addClass("full");
				break;	
			default:
				break;
		}
			
		// animate in
		clearTimeout(destroyTimer);
		switch(opts.animate) {
			case "fade":
				$("#notifyBox").stop().fadeIn(function() {
					destroyTimer = setTimeout(opts.destroyNotify,opts.duration);
				});
				break;
			case "slide":
				$("#notifyBox").stop().slideToggle(function() {
					destroyTimer = setTimeout(opts.destroyNotify,opts.duration);
				});
				break;
			default:
				break;	
		}
	}
});

// scroll to bottom
RT.plugins.scrollToBottom = function(el) {
	el.scrollTop = el.scrollHeight;
}

// liveCounter Widget
RT.plugins.liveCounter = function(ch,el) {
	RT.plugin(ch, function(e) {
		if(e.type == "command" && e.data.command == "onSubscribe") {
			el.innerHTML = e.data.count;
		}
	})
}