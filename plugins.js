// MAKE BADGE PLUGIN
	//top right bottom left
	// flash option
// MAKE WHOS ONLINE PHP plugin
// flash title bar

// order managment openjs grid live update
// standard background changing


// jQuery is not required for this plugin
RT.plugins.isTyping = function(o) {
	var isTyping = 0;
	var typingTimer = null;
	var notify;
	
	// listen for onTyping	
	RT.addCommandEvent(o.channel,"onTyping",function(e) {
		o.output.innerHTML = o.output.innerHTML + "<span id='typing'>"+e.identity+" is typing<span>";
	});
	
	// listen for finished Typing
	RT.addCommandEvent(o.channel,"doneTyping",function(e) {
		var it = document.getElementById("typing");
		if(it) o.output.removeChild(it);
	});

	// listen for keydown
	o.input.addEventListener("keydown",function(e) {
		if(e.keyCode != 13) {
		
			clearTimeout(typingTimer);
			
			// trigger that you are typing
			if(!isTyping) {
				isTyping = 1;
				RT.triggerEvent(o.channel,"onTyping",{},true);
			}	
			
			// trigger that you are done typing
			typingTimer = setTimeout(function() {
				isTyping = false;
				RT.triggerEvent(o.channel,"doneTyping");
			},1000);
			
		}				
	},"false");
	
	// when a message is received remove the istyping
	RT.addMessageEvent(o.channel,function(e) {
		var it = document.getElementById("typing");
		if(it) o.output.removeChild(it);
	});

}

// creates a badge for some container
// jQuery is required for this plugin
function RT_Badge(userOpts) {
	
	this.$badge = null;
	this.z = 0;
	this.opts = {
		container : $("body")[0],
		position : "topright",
		theme : "red",
		offsetX : -10,
		offsetY : -10,
		animate : "blink",
		shape : "round",
		x : 0
	};

	// add the badge to the page
	this.apply = function(userOpts) {
		this.opts = $.extend(this.opts,userOpts);
		var $container = $(this.opts.container);
		
		if(!this.$badge) {
			this.$badge = $("<div class='rt_badge'></div>");
			// determine where to put the badge
			var left = $container.offset().left;
			var top = $container.offset().top;
			this.$badge.css({
				left:left + $container.outerWidth() + this.opts.offsetX, 
				top:top + this.opts.offsetY
			});
			if(this.opts.shape == "square") this.$badge.addClass("square");
			$("body").append(this.$badge);
			
			// update badge info
			this.update(this.opts.x);
			
		}
				
	}
	// blink the object
	this.blink = function() {
		if(this.$badge.css("display") == "none") {
			this.$badge.stop().fadeIn().fadeOut().fadeIn();
		} else {
			this.$badge.stop().fadeOut().fadeIn();
		}
	}
	
	// show the object
	this.show = function() {
		switch(this.opts.animate) {
			case "blink":
				this.z ? this.blink() : this.hide();
				break;
			case "none":
			default:
				this.z ? this.$badge.show() : this.hide();
				break;
		}
	}
	
	// hide it
	this.hide = function() {
		if(this.$badge) {
			this.$badge.fadeOut();
		}
	}
	
	// update the text
	this.update = function(x) {
		this.opts = $.extend(this.opts,userOpts);
		if(this.$badge) {
			// allow for +1 and -1
			x = x.toString();
			var y = parseInt(x.substr(1,x.length));
			if(x.charAt(0) == "+") {
				this.z += y;
			} else if(x.charAt(0) == "-") {
				// protect against neg numbers
				this.z = this.z <= 0 ? 0 : this.z - y;
			} else {
				this.z = parseInt(x);
			}
			// set the text
			this.$badge.text(this.z);
			// show the badge
			this.show();
		 
		 // if there is no badge, add it
		 } else {
		 	this.apply({x : x});
		 }
	}
	
}


// create an instance of the badge plugin
RT.plugins.badge = function(userOpts) {
	var badge = new RT_Badge();
	badge.apply(userOpts);
	return badge;
}

// notify object that gets created when you call the plugin
function RT_Notify(userOpts) {
	
	this.$el = null;
	this.destroyTimer = null;
	this.opts = {
		theme : "google",
		duration : 2000,
		position : "bottomright",
		animate : "fade",
		title : "",
		msg : ""
	};
	
	this.die = function(msg) {
		clearTimeout(this.destroyNotify);
		this.$el.stop().remove();
	}
	
	// destroy the notify box
	this.destroyNotify = function(self) {
		// this needs to be the object
		self.$el.stop().fadeOut(function() {
			self.die();
		});
	}
	
	// show the notify box
	this.show = function(userOpts) {

		this.opts = $.extend(this.opts,userOpts);
		
		// clear any existing timer
		clearTimeout(this.destroyTimer);
		
		// create element if it doesnt exist
		if(!this.$el) {
			this.$el = $("\
				<div class='notifyBox'>\
					<div class='notifyTitle'></div>\
					<div class='notifyBody'></div>\
				</div>\
			");
			$("body").append(this.$el);
		}
		
		var self = this;
		
		// stay on hover
		this.$el.hover(function() {
			clearTimeout(self.destroyTimer);
		}, function() {
			self.destroyTimer = setTimeout(self.destroyNotify,self.opts.duration,self);
		});
		
		// put the text in
		this.$el.find(".notifyTitle").html(this.opts.title);
		this.$el.find(".notifyBody").html(this.opts.msg);
		
		// trick for icons
		if(!this.opts.title) {
			this.$el.find(".notifyTitle").css("float","left");
		} else {
			this.$el.find(".notifyTitle").css("float","none");
		}	
			
		// theme
		this.$el.addClass(this.opts.theme);	
			
		// refresh
		this.$el.css({
			"bottom":"auto",
			"top":"auto",
			"right":"auto",
			"left":"auto"
		});
		
		// place it
		switch(this.opts.position) {
			case "bottomright":
				this.$el.css({
					"bottom":0,
					"right":0
				});
				break;
			case "bottomleft":
				this.$el.css({
					"bottom":0,
					"left":0
				});
				break;
			case "topright":
				this.$el.css({
					"top":0,
					"right":0
				});
				break;
			case "topleft":
				this.$el.css({
					"top":0,
					"left":0
				});
				break;
			case "top":
				this.$el.css({
					"top":0,
					"left":0,
					"max-width":"none",
					"width":"100%",
					"margin":"0px"
				}).addClass("full");
				break;	
			case "bottom":
				this.$el.css({
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
		clearTimeout(this.destroyTimer);
		self = this;
		
		switch(this.opts.animate) {
			case "fade":
				this.$el.stop().fadeIn(function() {
					self.destroyTimer = setTimeout(self.destroyNotify,self.opts.duration,self);
				});
				break;	
			case "slide":
				this.$el.stop().slideToggle(function() {
					self.destroyTimer = setTimeout(self.destroyNotify,self.opts.duration,self);
				});
				break;
			default:
				break;	
		}
	}	
}

// create an instance of the notify plugin
RT.plugins.notify = function(userOpts) {
	var notify = new RT_Notify();
	notify.show(userOpts);
	return notify;
}

// scroll to bottom
RT.plugins.scrollToBottom = function(el) {
	el.scrollTop = el.scrollHeight;
}

// liveCounter Widget
RT.plugins.liveCounter = function(o) {
	RT.addCommandEvent(o.channel,"onSubscribe",function(e) {
		o.counter.innerHTML = e.data.count;
	})
}