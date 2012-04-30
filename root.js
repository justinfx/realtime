// Root Version SC.4.27.12
(function($) {	
	"use strict";
	
	window.Root = {
		DOM : null,					// the DOM object of this instance' element
		id : null,					// the ID on the DOM element
		_events : function() {},	// object of events for the DOM element
		_construct: function() {},	// construct method
		_init: function() {},		// method called on every instantiation
		proto: function () {                     // shortcut to parent object
            return Object.getPrototypeOf(this);
        },
        // determine if the parent element is root or not
        // used to make sure the hasOwnProperty(_construct) doesn't get called for definitions
        isRoot : function(parent) {
        	return !parent.proto().hasOwnProperty("DOM");
        },

		// generate random IDs on the fly
		generateId : function() {
			var r = Math.floor(Math.random()*99999999);
			return "root_"+r;
		},
		
		// helper on function to shorten syntax when using events
		// on the stored dom element for this instance
		// this.on(type,selector,method) || this.on(type,method)
		on : function() {
			var args = Array.prototype.slice.call(arguments);
			[this.DOM].on.apply([this.DOM],arguments);
		},
		
		// internal event handler
		// exectued from [HTMLElement].on('type','selector','callbcak')
		_on : function(els,type,selector,callback) {
			var instance = this,
				eventHandler = function(e) {
					console.log("event");
					var el = this,
					// get all matched elements
					// if no selection, use [element]
					results = (!selector) ? [el] : this.querySelectorAll(selector);
					
					console.log(results);
					
					// loop through matching elements and try to find e.target
					for(var i=0, l=results.length; i<l; i++) {
					
						// get this element to look for from the results
						var currEl = results[i],
							looking = e.target;

						// we need to now bubble up from e.target till we find our selector
						while(looking != currEl) {
							if(!looking) break;
							else looking = looking.parentNode;
						}
						
						// we've gone a round of looking, now check to see if the last element we 
						// broke from is our guy
						if(looking === currEl) instance[callback](looking,e);
	
					}
				}
			
			
			// loop through passed in elements, and for selectors,
			// expand into domElements, and for domElement, just push them
			var elements = [];
			for(var j=0;j<els.length;j++) {
				var element = els[j];
				
				// if its a string, loop and add
				if(typeof element === "string") {
					var results = document.querySelectorAll(element);
					for(var k=0;k<results.length;k++) {
						elements.push(results[k]);
					}	
					
				// dom element, just add
				} else if (element.nodeType && element.nodeType === 1) {
					elements.push(element);
				}
			}
			
			// apply events to matched domElements
			for(var l=0;l<elements.length;l++) {
				// add the event to the passed parent element
				var element = elements[l];
				console.log("adding event",element,type);
				element.addEventListener(type,function() {
					console.log("here");
				},true);
			}
			
		},
		// object inheritance method
		// takes an object to extend the class with, &| a dom element to use
		inherit: function(values,DOM) {
			
			// define the parent
			var parent = this;
			
			// Create a new object instance with "this" as the linked prototype
			var instance = Object.create(parent);
			
			// prototype the array object in the scope of this instance. heh.
			Array.prototype.on = function(type,sel,method) {
				var args = Array.prototype.slice.call(arguments);
				
				// if we only passed in the type and method, pass no selection
				if(args.length === 2) {
					method = sel; 
					sel = "";
				}
					
				// call internal on method in the scope of this instance
				instance._on(this,type,sel,method);
			}
			
			
			// if you passed in the DOM element, 
			if(DOM) values.DOM = DOM;
			
			// whenever we set this.DOM or instance.DOM
			// apply events
			var placeholder;
            Object.defineProperty(instance, "DOM", {
                get: function () { return placeholder; },
                set: function (DOM) {
                    // completes the setter
                    placeholder = DOM;
                    
                    // call events on the parent, not on the instances
					if(parent.hasOwnProperty("_events")) {
						instance._events();
					}
                    
                }
            });
			
			// Copy over properties from the old to the new
			// this triggers the above defineProperty
			for ( var prop in values ) {
				if(values) {
					instance[prop] = values[prop];
					// if one of the properties was DOM, set the instance on that
					if(prop === "DOM") {
						instance[prop].instance = instance;
					}
				}
			}

			// if the parent element has a constructor, call it on the instances
			// dont call if your parent is root
			if(parent.hasOwnProperty("_construct") && !this.isRoot(parent)) {
				parent._construct.apply(instance, null);
			}
			
			// if i have an _init function, call me
			if(instance.hasOwnProperty("_init")) {
				instance._init();
			}

			// return the new instance
			return instance;
		}
	};
	
	
})(window.jQuery);