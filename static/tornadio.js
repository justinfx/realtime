/******************************************************************
RealTime Instant Communication
Written By Sean Clark
April 15th 2011

DEPENDENCIES
socketIO required
http://cdn.socket.io/stable/socket.io.js
WEB_SOCKET_SWF_LOCATION = 'http://cdn.socket.io/stable/WebSocketMain.swf'
And the entire python + Tornado + ZeroMQ server

Normal Procedure:

1) In some global JS file define all of your channel functionality by
	creating a bunch of createChannel('channelName',{methods})
2) On page load - connect to realTime using RT.connect() and pass
	it your identity. Normally either null, or gotten from PHP or some
	other script that passes a user_id
3) Subscribe to some channels using RT.subscribe('channel')
	Subscribe on page load to some global channel if you want something
	always running.

******************************************************************/

WEB_SOCKET_SWF_LOCATION = 'http://cdn.socket.io/stable/WebSocketMain.swf';

var RT = {
	
	debugging:true,				// turn debugging on or off
	socket : null,				// the socketIO object
	channelMethods : {},		// each channel has its own object
	identity: null,
	options : {
		strip : false,			// regex to be replaced by blank
		htmlEntities : true,	// turn > and < into &lt; and &gt;
		dateFormat : "longTime",// date format http://blog.stevenlevithan.com/archives/date-time-format
		daysToSave : 1,			// days to save channels and identity as cookies
		port:8001,
		resource:"realtime",
		channelsCookie : "realTime_channels",
		identityCookie : "realTime_identity"
	},
	
	
	/******************************************************************
	Method:	connect
	Input:	identity [String]
	Input:	options [Object]
	Input: 	callback [Function]

	Connects to socketIO.  You must call this to start the system. You
	pass it a unique identity of who you are. You can also pass it an 
	options object which overwrites the defaults in this.options. Lastly
	you can pass a callback which will get execute on connect. Alternatively
	you can listen for onInit. Note the callback here is only when socketiO
	is connected, not zmq, if you want the final connection status ready, then
	wait for onInit
	
	******************************************************************/
	connect : function(identity,options,callback) {
		// create new socketIO Object
		this.socket = new io.Socket(window.location.hostname, {
        	port: (options && options.port) || this.options.port, 
        	rememberTransport: false,
        	resource: (options && options.resource) || this.options.resource
        });
        // connect to socketIO
        this.socket.connect();
        
        // extend options
		for(i in options) {
			this.options[i] = options[i];
		}
        
        // save identity
        this.saveIdentity(identity);
        
        this.debug("Connected: ",this);
        this.debug("savedChannels: ",this.getSavedChannles());
        
        // create init message
        var init = {
        	type : "command",
			identity : identity,
			data : {
				command : "init",
				options : {
					channels : this.getSavedChannles()
				}	
			}
        }
        
        // pass init to server
        this.socket.send(init);
        this.debug("init: ",init);
        
        // if callback was passed
        if(callback) {
        	this.socket.addEvent('connect',callback);
        }
        
        /******************************************************************
		SocketIO.addEvent('message')
		Input:	[Function]

		Whenever ANY message is received by SocketIO this gets called. The
		object that is passed contains all the needed data to parse how this
		message is handled. There are type 'message' and type 'command'.
		******************************************************************/
        this.socket.addEvent('message', function(json) {
        	
        	// make sure its an object
        	if(typeof(json) != "object") json = JSON.parse(json);
        	
        	RT.debug("Incoming Message ["+json.channel+"]: ",json);
			
			// handle different types of messages
			if(json.type && json.channel) {
				var channel = json.channel;
				var channelObj = RT.channelMethods[channel];
				
				switch(json.type) {
					/*********************************************************
					For messages, we just make sure the onReceive method
					is set. Then we call whatever the developer defined to
					happen on the onReceive method of the createChannel call.
					**********************************************************/
					case "message":
						if(channelObj._hasMethod("onReceive")) {
							// if we need to regex some stuff out
							if(RT.options.strip || RT.options.htmlEntities) {
								// turn the object into a string while regexing the string values
								var stripped = JSON.stringify(json.data, function(key,value) {
									if(typeof value === 'string') {
										if(RT.options.strip) {
											return value.replace(RT.options.strip,"");
										} else if (RT.options.htmlEntities) {
											return value.replace(/</,"&lt;").replace(/>/,"&gt;");
										}	
									}
									// always return something
									return value;
								});
								// turn the regexed string back into an object
								json.data = JSON.parse(stripped);
							}
							
							// format the time
							json.timestamp = RT.formatDate(json.timestamp);
							
							// call the user defined receive method and pass just the data
							channelObj.onReceive(json);
							
							RT.debug("Calling Channel Method: ",channel);
						}
						break
					/*********************************************************
					For commands, we make sure the command has been defined by
					the developer in the createChannel call. This will handle
					things like onConnect and onDisconnect
					**********************************************************/
					case "command":
						
						if(channelObj._hasMethod(json.data.command)) {
							
							// determine if your supposed to do this on yourself
							if(!json.data.notMe || (json.identity != RT.identity)) {
							
								// if a date format is set, convert timestamp
								json.timestamp = RT.formatDate(json.timestamp);
								
								// call the channelMethod with a new abstracted object
								channelObj[json.data.command]({
									identity : json.identity,
									timestamp : json.timestamp
								});
								RT.debug("Command: ["+channel+"]["+json.data.command+"]",json);
							}	
						}
						break;
					default:
						break;
				}
			}
        }); 
	},
	
	/******************************************************************
	Method: saveIdentity
	Input:	[String]
	Output:	None
	
	Saves the identity into the browser cookie for identity for the
	time specified in the options daysToSave
	******************************************************************/
	saveIdentity : function(identity) {
		if(identity && typeof(identity) == "string") {
			this.identity = identity;
			setCookie(this.options.identityCookie,identity,this.options.daysToSave);
		}	
	},
	
	/******************************************************************
	Method: clearIdentity
	Input:	None
	Output:	None
	
	Destroys the identity property and the cookie
	******************************************************************/
	clearIdentity : function() {
		setCookie(this.options.identityCookie,"",0);
		this.identity = null;
	},
	
	/******************************************************************
	Method: saveChannel
	Input:	[String]
	Output:	None
	
	Will add the given channel to the array of channels and save that
	to the browser cookie.
	******************************************************************/
	saveChannel : function(channel) {
		// get current channels
		var channels = this.getSavedChannles();
		if(channels) {
			// add this channel to array (if its not there already)
			if(!inArray(channel,channels)) {
				channels.push(channel);
			}	
			// re-store the channels
			setCookie(this.options.channelsCookie,channels.join(","),this.options.daysToSave);
		}	
	},
	
	/******************************************************************
	Method: unSaveChannel
	Input:	[String]
	Output:	None
	
	Will remove the given channel from the array of saved channels
	inside the browser cookie
	******************************************************************/
	unSaveChannel : function(channel) {
		var channels = this.getSavedChannles();
		if(channels) {
			// find out where it is in the array
			var index = channels.indexOf(channel);
			// if found - remove
			if(index != -1) channels.splice(index,1);
			// re save
			setCookie(this.options.channelsCookie,channels.join(","),this.options.daysToSave);
		}
	},
	
	/******************************************************************
	Method: getSavedChannels
	Input:	None
	Output:	[Array] the saved channels
	
	Will find in the browser cookie the saved channels for your client
	returns an array of those or null if none found
	******************************************************************/
	getSavedChannles : function() {
		var channels = getCookie(this.options.channelsCookie);
		if(channels) {
			return channels.split(",");
		} else {
			return false;
		}	
	},
	
	
	/******************************************************************
	Method: subscribe
	Input:	channels [Array]|[String]
	Input:	identity [String]
	Output:	None
	
	Makes your client start listening for messages along a specific
	channel or channels. This function sends the subscribe command to
	the server.  data is used to pass any extra data like time, identity
	dom IDs and so forth. This data is then available in any of the 
	channelMethods methods that are called with type command.
	******************************************************************/
	subscribe : function(channels) {
		
		// connect if not connected
		if(!this.socket) this.connect();
		
		// if you just passed a single channel
		if(typeof(channels) == "string") {
			// now make it an array for the rest of the time
			channels = new Array(channels);
		}
		for(i in channels) {
			var channel = channels[i];
			// send the subscribe message to tornado
			this.debug("Subscription: ",channel);
			this.socket.send({
				type : "command",
				channel : channel,
				data : {
					command : "subscribe"
				}
			});
			// save this channel
			this.saveChannel(channel);
		}
	},
	
	/******************************************************************
	Method: unsubscribe
	Input:	channels [Array]|[String]
	Output:	None
	
	takes 1 or more channels, and sends the unsubscribe command to 
	the server.
	******************************************************************/
	unsubscribe : function(channels) {
		if(typeof(channels) == "string") {
			channels = new Array(channels);
		}
		for(i in channels) {
			var channel = channels[i];
			// send the subscribe message to tornado
			this.debug("Subscription: ",channel);
			this.socket.send({
				type : "command",
				channel : channel,
				data : {
					command : "unsubscribe"
				}
			});
			// unsave this channel
			this.unSaveChannel(channel);
		}
	},
	
	
	/******************************************************************
	Method: publish
	Input:	channel [String]
	Input: 	msg [String]|[Object]
	Output:	None
	
	Sends a message to the server along the specefied channel.
	The msg is the same msg that is received in the developer defined
	handler.  So its up to the developer to determine how to handle the
	msg.  Will connect to SocketIO if no connection exists
	******************************************************************/
	publish : function(channel,msg) {

		if(!this.socket) this.connect();
		this.debug("Publishing ["+channel+"]: ",msg);
		this.socket.send({
			type : "message",
			channel : channel,
			data : {
				msg : msg
			}
		});
		
		if(this.channelMethods[channel]) {
			// call the onSend handler for this channel if on existed
			if(typeof(this.channelMethods[channel].onSend) == "function") {
				this.channelMethods[channel].onSend(msg);
			}
		} else {
			this.debug("Channel "+channel+" not created");
		}
	},
	
	
	
	/******************************************************************
	Method: createChannel
	Input:	channel [String]
	Input:	methods [Object]
				onReceive [Function]
				onSend [Function]
				onSubscribe [Function]
				onUnsubscribe [Function]
				onDisconnect [Function]
	Output:	None
	
	These are the "plugins" that the developer defines to handle what
	happens when certain events occur.  This will take an object of
	the methods which are pre-defined here.  This function simply
	defines the methods on the channel index of the channelMethods object
	so they can later be called.
	******************************************************************/
	createChannel : function(channel,methods) {
		
		// append all passed methods to this channel method
		this.channelMethods[channel] = methods;
		
		// add an extra method for testing for other methods
		this.channelMethods[channel]["_hasMethod"] = function(method) {
			return typeof(this[method]) == "function"
		}
		
		// add a quick way to get to triggerEvent
		this.channelMethods[channel]["_triggerEvent"] = function(event,options,notMe) {
			this.triggerEvent(channel, event, options, notMe);
		}
		
		this.debug("Creating Channel: ",channel)
	},
	
	/******************************************************************
	Method: triggerEvent
	Input:	channel [String]
	Input:	event [String]
	Input:	options [Object]
	Input:	notMe [Bool]
	Output:	None
	
	This creates an arbitrary command to the sytem that will be passed
	through to all users. It sends the event along the given channel with
	data.options as the passed options object.  If you pass notMe as 
	true, then the event will not fire for you, only everyone else
	******************************************************************/
	triggerEvent : function(channel, event, options, notMe) {
		this.debug("triggerEvent: ",event);
		this.socket.send({
			type : "command",
			channel : channel,
			data : {
				command : event,
				options : options,
				notMe : notMe
			}
		});
	},

	/******************************************************************
	Method: debug
	Input:	data1 [Any]
	Input:	data2 [Any]
	Output:	To Javascript Console
	
	debugging console
	******************************************************************/
	debug : function(data1,data2) {
		try {
			if(this.debugging && window.console && console.log) {
				console.log(data1,data2);
			}
		} catch(e) {}	
	},
	
	/******************************************************************
	Method: formatDate
	Input:	timestamp [String]
	Output:	formattted datetime
	
	Given any timestamp string, will use the dateFormat class to format
	the date based on the realTime option dateFormat
	******************************************************************/
	formatDate : function(timestamp) {
		if(this.options.dateFormat) {
			var d = new Date(Date(timestamp));
			return d.format(this.options.dateFormat);
		} else {
			return timestamp;
		}	
	}
	
}

/******************************************************************
Function: setCookie
Input:	c_name [String]
Input:	value [String]
Input:	exdays [Int]
Output:	None

Will set a browser cookie for the specefied number of days
******************************************************************/
function setCookie(c_name,value,exdays) {
	var exdate=new Date();
	exdate.setDate(exdate.getDate() + exdays);
	var c_value=escape(value) + ((exdays==null) ? "" : "; expires="+exdate.toUTCString());
	document.cookie=c_name + "=" + c_value;
}

/******************************************************************
Function: getCoookie
Input:	c_name [String]
Output:	[String] the cookie value

Will return null or the value of the give cookie
******************************************************************/
function getCookie(c_name) {
	var i,x,y,ARRcookies=document.cookie.split(";");
	for (i=0;i<ARRcookies.length;i++) {
		x=ARRcookies[i].substr(0,ARRcookies[i].indexOf("="));
		y=ARRcookies[i].substr(ARRcookies[i].indexOf("=")+1);
		x=x.replace(/^\s+|\s+$/g,"");
		if (x==c_name) {
			return unescape(y);
		}
	}
}

/******************************************************************
Function: inArray
Input:	needle [String]
Input:	haystack [Array]
Input:	argStrict [Bool]
Output:	[Bool] weather the value exists in the array or not

Will search the array for the value. If you want the value to be
strict, as in 1 != "1" then pass true for argStrict
******************************************************************/
function inArray (needle, haystack, argStrict) {
	var key = '',
    strict = !! argStrict;

    if (strict) {
        for (key in haystack) {
            if (haystack[key] === needle) return true;
        }
    } else {
        for (key in haystack) {
            if (haystack[key] == needle) return true;
        }
    }
    return false;
}

/*
 * Date Format 1.2.3
 * (c) 2007-2009 Steven Levithan <stevenlevithan.com>
 * MIT license
 *
 * Includes enhancements by Scott Trenda <scott.trenda.net>
 * and Kris Kowal <cixar.com/~kris.kowal/>
 *
 * Accepts a date, a mask, or a date and a mask.
 * Returns a formatted version of the given date.
 * The date defaults to the current date/time.
 * The mask defaults to dateFormat.masks.default.
 * http://blog.stevenlevithan.com/archives/date-time-format
 */
var dateFormat = function () {
	var	token = /d{1,4}|m{1,4}|yy(?:yy)?|([HhMsTt])\1?|[LloSZ]|"[^"]*"|'[^']*'/g,
		timezone = /\b(?:[PMCEA][SDP]T|(?:Pacific|Mountain|Central|Eastern|Atlantic) (?:Standard|Daylight|Prevailing) Time|(?:GMT|UTC)(?:[-+]\d{4})?)\b/g,
		timezoneClip = /[^-+\dA-Z]/g,
		pad = function (val, len) {
			val = String(val);
			len = len || 2;
			while (val.length < len) val = "0" + val;
			return val;
		};

	// Regexes and supporting functions are cached through closure
	return function (date, mask, utc) {
		var dF = dateFormat;

		// You can't provide utc if you skip other args (use the "UTC:" mask prefix)
		if (arguments.length == 1 && Object.prototype.toString.call(date) == "[object String]" && !/\d/.test(date)) {
			mask = date;
			date = undefined;
		}

		// Passing date through Date applies Date.parse, if necessary
		date = date ? new Date(date) : new Date;
		if (isNaN(date)) throw SyntaxError("invalid date");

		mask = String(dF.masks[mask] || mask || dF.masks["default"]);

		// Allow setting the utc argument via the mask
		if (mask.slice(0, 4) == "UTC:") {
			mask = mask.slice(4);
			utc = true;
		}

		var	_ = utc ? "getUTC" : "get",
			d = date[_ + "Date"](),
			D = date[_ + "Day"](),
			m = date[_ + "Month"](),
			y = date[_ + "FullYear"](),
			H = date[_ + "Hours"](),
			M = date[_ + "Minutes"](),
			s = date[_ + "Seconds"](),
			L = date[_ + "Milliseconds"](),
			o = utc ? 0 : date.getTimezoneOffset(),
			flags = {
				d:    d,
				dd:   pad(d),
				ddd:  dF.i18n.dayNames[D],
				dddd: dF.i18n.dayNames[D + 7],
				m:    m + 1,
				mm:   pad(m + 1),
				mmm:  dF.i18n.monthNames[m],
				mmmm: dF.i18n.monthNames[m + 12],
				yy:   String(y).slice(2),
				yyyy: y,
				h:    H % 12 || 12,
				hh:   pad(H % 12 || 12),
				H:    H,
				HH:   pad(H),
				M:    M,
				MM:   pad(M),
				s:    s,
				ss:   pad(s),
				l:    pad(L, 3),
				L:    pad(L > 99 ? Math.round(L / 10) : L),
				t:    H < 12 ? "a"  : "p",
				tt:   H < 12 ? "am" : "pm",
				T:    H < 12 ? "A"  : "P",
				TT:   H < 12 ? "AM" : "PM",
				Z:    utc ? "UTC" : (String(date).match(timezone) || [""]).pop().replace(timezoneClip, ""),
				o:    (o > 0 ? "-" : "+") + pad(Math.floor(Math.abs(o) / 60) * 100 + Math.abs(o) % 60, 4),
				S:    ["th", "st", "nd", "rd"][d % 10 > 3 ? 0 : (d % 100 - d % 10 != 10) * d % 10]
			};

		return mask.replace(token, function ($0) {
			return $0 in flags ? flags[$0] : $0.slice(1, $0.length - 1);
		});
	};
}();

// Some common format strings
dateFormat.masks = {
	"default":      "ddd mmm dd yyyy HH:MM:ss",
	shortDate:      "m/d/yy",
	mediumDate:     "mmm d, yyyy",
	longDate:       "mmmm d, yyyy",
	fullDate:       "dddd, mmmm d, yyyy",
	shortTime:      "h:MM TT",
	mediumTime:     "h:MM:ss TT",
	longTime:       "h:MM:ss TT Z",
	isoDate:        "yyyy-mm-dd",
	isoTime:        "HH:MM:ss",
	isoDateTime:    "yyyy-mm-dd'T'HH:MM:ss",
	isoUtcDateTime: "UTC:yyyy-mm-dd'T'HH:MM:ss'Z'"
};

// Internationalization strings
dateFormat.i18n = {
	dayNames: [
		"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat",
		"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"
	],
	monthNames: [
		"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
		"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"
	]
};

// For convenience...
Date.prototype.format = function (mask, utc) {
	return dateFormat(this, mask, utc);
};