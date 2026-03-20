// SES QuickJS Polyfills — must run BEFORE ses.umd.js is loaded.
// Proven in experiments/lifecycle/QUICKJS_SES_POLYFILLS.md (79 experiments passing).
//
// Two issues in QuickJS that prevent SES lockdown():
// 1. Missing console methods that SES's tameConsole expects
// 2. Iterator prototype faux data properties that crash tameFauxDataProperty

// 1. Console methods
if (typeof console === "undefined") globalThis.console = {};
if (!console._times) console._times = {};
["log","warn","error","info","debug","time","timeEnd","timeLog",
 "group","groupEnd","groupCollapsed","assert","count","countReset",
 "dir","dirxml","table","trace","clear","profile","profileEnd",
 "timeStamp"].forEach(function(m) {
    if (!console[m]) console[m] = function() {};
});

// 2. Performance API
if (typeof performance === "undefined") {
    globalThis.performance = { now: function() { return Date.now(); } };
}

// 3. Node.js stubs
if (typeof process === "undefined") {
    globalThis.process = { env: {}, versions: {} };
}
if (typeof SharedArrayBuffer === "undefined") {
    globalThis.SharedArrayBuffer = ArrayBuffer;
}

// 4. Iterator prototype fix — convert faux data properties to real data properties.
// QuickJS's Iterator Helpers expose constructor and Symbol.toStringTag as accessors
// with setters that crash when SES's tameFauxDataProperty calls them on { __proto__: null }.
(function() {
    try {
        var ai = [][Symbol.iterator]();
        var ip = Object.getPrototypeOf(Object.getPrototypeOf(ai));
        if (ip) {
            var cd = Object.getOwnPropertyDescriptor(ip, 'constructor');
            if (cd && cd.get && !('value' in cd)) {
                Object.defineProperty(ip, 'constructor', {
                    value: cd.get.call(ip), writable: true,
                    enumerable: false, configurable: true
                });
            }
            var td = Object.getOwnPropertyDescriptor(ip, Symbol.toStringTag);
            if (td && td.get && !('value' in td)) {
                Object.defineProperty(ip, Symbol.toStringTag, {
                    value: td.get.call(ip), writable: false,
                    enumerable: false, configurable: true
                });
            }
        }
    } catch(e) { /* ignore on engines without Iterator Helpers */ }
})();
