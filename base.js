// Base Version SC.4.22.12
(function($) {	
	"use strict";
	
	/**
	EVENTS NOTE.
		Right now there are a few ways to define events. 
		Every instance has a method called _on.  Which applies an event to the stored DOM object.
		You can define a bunch of these kinds of events in the _events property, which will get executed upon instantiation.
		You can also define them anywhere else, the above property is just a place for them to go.
		You can also call _on on a DOM element, to attach an event to that guy, which will call a method on the instnace in which you defined the event
		You can also do the above on a jQuery collection.
		
	**/
	
	// apply event directly to DOM element
	// apparantly this is no recommended?
	window.HTMLElement.prototype._on = function(instance,type,method) {
		instance._on(type,"",method,this);
	};
	
	// jQuery way of adding event directly to DOM element
	$.fn._on = function(instance,type,method) {
		return this.each(function() {
			instance._on(type,"",method,this);
		});
	}
	
	window.Base = {
		DOM : null,					// the DOM object of this instance' element
		id : null,					// the ID on the DOM element
		_events : function() {},	// object of events for the DOM element
		_construct: function() {},	// construct method
		_init: function() {},		// method called on every instantiation
		proto: function () {                     // shortcut to parent object
            return Object.getPrototypeOf(this);
        },
        // determine if the parent element is base or not
        // used to make sure the hasOwnProperty(_construct) doesn't get called for definitions
        isBase : function(parent) {
        	return !parent.proto().hasOwnProperty("DOM");
        },

		// generate random IDs on the fly
		generateId : function() {
			var r = Math.floor(Math.random()*99999999);
			return "base_"+r;
		},
		// newInstance creates a small object instance and ties the DOM element to the instance
		// you can ghost with an object you wish to extend as well
		instance : function(DOM,obj) {
			// set the dom element on the object to be created
			obj.DOM = DOM;
			// create this instance
			var instance = this.inherit(obj);
			// return it
			return instance;
		},
		_on : function(type,sel,method,context) {
			if(!context) context = this.DOM;
			var self = this;
	
			$(context).on(type,sel,function(e) {
				self[method].call(self,this,e);
			});
		},
		// object inheritance method
		// takes an object to extend the class with, &| a dom element to use
		inherit: function(values,DOM) {
			
			// define the parent
			var parent = this;
			
			// Create a new object instance with "this" as the linked prototype
			var instance = Object.create(parent);
			
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
			// dont call if your parent is base
			if(parent.hasOwnProperty("_construct") && !this.isBase(parent)) {
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


Object.size = function(obj) {
    var size = 0, key;
    for (key in obj) {
        if (obj.hasOwnProperty(key)) size++;
    }
    return size;
};