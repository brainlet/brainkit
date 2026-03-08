package picomatch

// extglobs_temp_part1_test.go — Ported from picomatch/test/extglobs-temp.js lines 1-600
// Some of tests were converted from bash 4.3, 4.4, and minimatch unit tests.
// This is called "temp" as a reminder to reorganize these tests and remove duplicates.

import (
	"testing"
)

func TestExtglobsTempPart1(t *testing.T) {
	win := &Options{Windows: true}

	t.Run("bash", func(t *testing.T) {
		t.Run("should match extended globs from the bash spec", func(t *testing.T) {
			// !(foo)
			assertMatch(t, true, "bar", "!(foo)", win)       // extglobs-temp.js:14
			assertMatch(t, true, "f", "!(foo)", win)         // extglobs-temp.js:15
			assertMatch(t, true, "fa", "!(foo)", win)        // extglobs-temp.js:16
			assertMatch(t, true, "fb", "!(foo)", win)        // extglobs-temp.js:17
			assertMatch(t, true, "ff", "!(foo)", win)        // extglobs-temp.js:18
			assertMatch(t, true, "fff", "!(foo)", win)       // extglobs-temp.js:19
			assertMatch(t, true, "fo", "!(foo)", win)        // extglobs-temp.js:20
			assertMatch(t, false, "foo", "!(foo)", win)      // extglobs-temp.js:21
			assertMatch(t, false, "foo/bar", "!(foo)", win)  // extglobs-temp.js:22
			assertMatch(t, false, "foo/bar", "!(foo)/*", win) // extglobs-temp.js:23
			assertMatch(t, true, "foobar", "!(foo)", win)    // extglobs-temp.js:24
			assertMatch(t, true, "foot", "!(foo)", win)      // extglobs-temp.js:25
			assertMatch(t, true, "foox", "!(foo)", win)      // extglobs-temp.js:26
			assertMatch(t, true, "o", "!(foo)", win)         // extglobs-temp.js:27
			assertMatch(t, true, "of", "!(foo)", win)        // extglobs-temp.js:28
			assertMatch(t, true, "ooo", "!(foo)", win)       // extglobs-temp.js:29
			assertMatch(t, true, "ox", "!(foo)", win)        // extglobs-temp.js:30
			assertMatch(t, true, "x", "!(foo)", win)         // extglobs-temp.js:31
			assertMatch(t, true, "xx", "!(foo)", win)        // extglobs-temp.js:32

			// !(!(foo))
			assertMatch(t, false, "bar", "!(!(foo))", win)     // extglobs-temp.js:34
			assertMatch(t, false, "f", "!(!(foo))", win)       // extglobs-temp.js:35
			assertMatch(t, false, "fa", "!(!(foo))", win)      // extglobs-temp.js:36
			assertMatch(t, false, "fb", "!(!(foo))", win)      // extglobs-temp.js:37
			assertMatch(t, false, "ff", "!(!(foo))", win)      // extglobs-temp.js:38
			assertMatch(t, false, "fff", "!(!(foo))", win)     // extglobs-temp.js:39
			assertMatch(t, false, "fo", "!(!(foo))", win)      // extglobs-temp.js:40
			assertMatch(t, true, "foo", "!(!(foo))", win)      // extglobs-temp.js:41
			assertMatch(t, true, "foo/bar", "!(!(bar)/baz)", win) // extglobs-temp.js:42
			assertMatch(t, false, "foo/bar", "!(!(foo))", win) // extglobs-temp.js:43
			assertMatch(t, false, "foobar", "!(!(foo))", win)  // extglobs-temp.js:44
			assertMatch(t, false, "foot", "!(!(foo))", win)    // extglobs-temp.js:45
			assertMatch(t, false, "foox", "!(!(foo))", win)    // extglobs-temp.js:46
			assertMatch(t, false, "o", "!(!(foo))", win)       // extglobs-temp.js:47
			assertMatch(t, false, "of", "!(!(foo))", win)      // extglobs-temp.js:48
			assertMatch(t, false, "ooo", "!(!(foo))", win)     // extglobs-temp.js:49
			assertMatch(t, false, "ox", "!(!(foo))", win)      // extglobs-temp.js:50
			assertMatch(t, false, "x", "!(!(foo))", win)       // extglobs-temp.js:51
			assertMatch(t, false, "xx", "!(!(foo))", win)      // extglobs-temp.js:52

			// !(!(!(foo)))
			assertMatch(t, true, "bar", "!(!(!(foo)))", win)      // extglobs-temp.js:54
			assertMatch(t, true, "f", "!(!(!(foo)))", win)        // extglobs-temp.js:55
			assertMatch(t, true, "fa", "!(!(!(foo)))", win)       // extglobs-temp.js:56
			assertMatch(t, true, "fb", "!(!(!(foo)))", win)       // extglobs-temp.js:57
			assertMatch(t, true, "ff", "!(!(!(foo)))", win)       // extglobs-temp.js:58
			assertMatch(t, true, "fff", "!(!(!(foo)))", win)      // extglobs-temp.js:59
			assertMatch(t, true, "fo", "!(!(!(foo)))", win)       // extglobs-temp.js:60
			assertMatch(t, false, "foo", "!(!(!(foo)))", win)     // extglobs-temp.js:61
			assertMatch(t, false, "foo/bar", "!(!(!(foo)))", win) // extglobs-temp.js:62
			assertMatch(t, true, "foobar", "!(!(!(foo)))", win)   // extglobs-temp.js:63
			assertMatch(t, true, "foot", "!(!(!(foo)))", win)     // extglobs-temp.js:64
			assertMatch(t, true, "foox", "!(!(!(foo)))", win)     // extglobs-temp.js:65
			assertMatch(t, true, "o", "!(!(!(foo)))", win)        // extglobs-temp.js:66
			assertMatch(t, true, "of", "!(!(!(foo)))", win)       // extglobs-temp.js:67
			assertMatch(t, true, "ooo", "!(!(!(foo)))", win)      // extglobs-temp.js:68
			assertMatch(t, true, "ox", "!(!(!(foo)))", win)       // extglobs-temp.js:69
			assertMatch(t, true, "x", "!(!(!(foo)))", win)        // extglobs-temp.js:70
			assertMatch(t, true, "xx", "!(!(!(foo)))", win)       // extglobs-temp.js:71

			// !(!(!(!(foo))))
			assertMatch(t, false, "bar", "!(!(!(!(foo))))", win)      // extglobs-temp.js:73
			assertMatch(t, false, "f", "!(!(!(!(foo))))", win)        // extglobs-temp.js:74
			assertMatch(t, false, "fa", "!(!(!(!(foo))))", win)       // extglobs-temp.js:75
			assertMatch(t, false, "fb", "!(!(!(!(foo))))", win)       // extglobs-temp.js:76
			assertMatch(t, false, "ff", "!(!(!(!(foo))))", win)       // extglobs-temp.js:77
			assertMatch(t, false, "fff", "!(!(!(!(foo))))", win)      // extglobs-temp.js:78
			assertMatch(t, false, "fo", "!(!(!(!(foo))))", win)       // extglobs-temp.js:79
			assertMatch(t, true, "foo", "!(!(!(!(foo))))", win)       // extglobs-temp.js:80
			assertMatch(t, false, "foo/bar", "!(!(!(!(foo))))", win)  // extglobs-temp.js:81
			assertMatch(t, false, "foot", "!(!(!(!(foo))))", win)     // extglobs-temp.js:82
			assertMatch(t, false, "o", "!(!(!(!(foo))))", win)        // extglobs-temp.js:83
			assertMatch(t, false, "of", "!(!(!(!(foo))))", win)       // extglobs-temp.js:84
			assertMatch(t, false, "ooo", "!(!(!(!(foo))))", win)      // extglobs-temp.js:85
			assertMatch(t, false, "ox", "!(!(!(!(foo))))", win)       // extglobs-temp.js:86
			assertMatch(t, false, "x", "!(!(!(!(foo))))", win)        // extglobs-temp.js:87
			assertMatch(t, false, "xx", "!(!(!(!(foo))))", win)       // extglobs-temp.js:88

			// !(!(foo))*
			assertMatch(t, false, "bar", "!(!(foo))*", win)    // extglobs-temp.js:90
			assertMatch(t, false, "f", "!(!(foo))*", win)      // extglobs-temp.js:91
			assertMatch(t, false, "fa", "!(!(foo))*", win)     // extglobs-temp.js:92
			assertMatch(t, false, "fb", "!(!(foo))*", win)     // extglobs-temp.js:93
			assertMatch(t, false, "ff", "!(!(foo))*", win)     // extglobs-temp.js:94
			assertMatch(t, false, "fff", "!(!(foo))*", win)    // extglobs-temp.js:95
			assertMatch(t, false, "fo", "!(!(foo))*", win)     // extglobs-temp.js:96
			assertMatch(t, true, "foo", "!(!(foo))*", win)     // extglobs-temp.js:97
			assertMatch(t, true, "foobar", "!(!(foo))*", win)  // extglobs-temp.js:98
			assertMatch(t, true, "foot", "!(!(foo))*", win)    // extglobs-temp.js:99
			assertMatch(t, true, "foox", "!(!(foo))*", win)    // extglobs-temp.js:100
			assertMatch(t, false, "o", "!(!(foo))*", win)      // extglobs-temp.js:101
			assertMatch(t, false, "of", "!(!(foo))*", win)     // extglobs-temp.js:102
			assertMatch(t, false, "ooo", "!(!(foo))*", win)    // extglobs-temp.js:103
			assertMatch(t, false, "ox", "!(!(foo))*", win)     // extglobs-temp.js:104
			assertMatch(t, false, "x", "!(!(foo))*", win)      // extglobs-temp.js:105
			assertMatch(t, false, "xx", "!(!(foo))*", win)     // extglobs-temp.js:106

			// !(f!(o))
			assertMatch(t, true, "bar", "!(f!(o))", win)       // extglobs-temp.js:108
			assertMatch(t, false, "f", "!(f!(o))", win)        // extglobs-temp.js:109
			assertMatch(t, false, "fa", "!(f!(o))", win)       // extglobs-temp.js:110
			assertMatch(t, false, "fb", "!(f!(o))", win)       // extglobs-temp.js:111
			assertMatch(t, false, "ff", "!(f!(o))", win)       // extglobs-temp.js:112
			assertMatch(t, false, "fff", "!(f!(o))", win)      // extglobs-temp.js:113
			assertMatch(t, true, "fo", "!(f!(o))", win)        // extglobs-temp.js:114
			assertMatch(t, true, "foo", "!(!(foo))", win)      // extglobs-temp.js:115
			assertMatch(t, false, "foo", "!(f)!(o)!(o)", win)  // extglobs-temp.js:116
			assertMatch(t, true, "foo", "!(fo)", win)          // extglobs-temp.js:117
			assertMatch(t, true, "foo", "!(f!(o)*)", win)      // extglobs-temp.js:118
			assertMatch(t, false, "foo", "!(f!(o))", win)      // extglobs-temp.js:119
			assertMatch(t, false, "foo/bar", "!(f!(o))", win)  // extglobs-temp.js:120
			assertMatch(t, false, "foobar", "!(f!(o))", win)   // extglobs-temp.js:121
			assertMatch(t, true, "o", "!(f!(o))", win)         // extglobs-temp.js:122
			assertMatch(t, true, "of", "!(f!(o))", win)        // extglobs-temp.js:123
			assertMatch(t, true, "ooo", "!(f!(o))", win)       // extglobs-temp.js:124
			assertMatch(t, true, "ox", "!(f!(o))", win)        // extglobs-temp.js:125
			assertMatch(t, true, "x", "!(f!(o))", win)         // extglobs-temp.js:126
			assertMatch(t, true, "xx", "!(f!(o))", win)        // extglobs-temp.js:127

			// !(f(o))
			assertMatch(t, true, "bar", "!(f(o))", win)        // extglobs-temp.js:129
			assertMatch(t, true, "f", "!(f(o))", win)          // extglobs-temp.js:130
			assertMatch(t, true, "fa", "!(f(o))", win)         // extglobs-temp.js:131
			assertMatch(t, true, "fb", "!(f(o))", win)         // extglobs-temp.js:132
			assertMatch(t, true, "ff", "!(f(o))", win)         // extglobs-temp.js:133
			assertMatch(t, true, "fff", "!(f(o))", win)        // extglobs-temp.js:134
			assertMatch(t, false, "fo", "!(f(o))", win)        // extglobs-temp.js:135
			assertMatch(t, true, "foo", "!(f(o))", win)        // extglobs-temp.js:136
			assertMatch(t, false, "foo/bar", "!(f(o))", win)   // extglobs-temp.js:137
			assertMatch(t, true, "foobar", "!(f(o))", win)     // extglobs-temp.js:138
			assertMatch(t, true, "foot", "!(f(o))", win)       // extglobs-temp.js:139
			assertMatch(t, true, "foox", "!(f(o))", win)       // extglobs-temp.js:140
			assertMatch(t, true, "o", "!(f(o))", win)          // extglobs-temp.js:141
			assertMatch(t, true, "of", "!(f(o))", win)         // extglobs-temp.js:142
			assertMatch(t, true, "ooo", "!(f(o))", win)        // extglobs-temp.js:143
			assertMatch(t, true, "ox", "!(f(o))", win)         // extglobs-temp.js:144
			assertMatch(t, true, "x", "!(f(o))", win)          // extglobs-temp.js:145
			assertMatch(t, true, "xx", "!(f(o))", win)         // extglobs-temp.js:146

			// !(f) — first block
			assertMatch(t, true, "bar", "!(f)", win)           // extglobs-temp.js:148
			assertMatch(t, false, "f", "!(f)", win)            // extglobs-temp.js:149
			assertMatch(t, true, "fa", "!(f)", win)            // extglobs-temp.js:150
			assertMatch(t, true, "fb", "!(f)", win)            // extglobs-temp.js:151
			assertMatch(t, true, "ff", "!(f)", win)            // extglobs-temp.js:152
			assertMatch(t, true, "fff", "!(f)", win)           // extglobs-temp.js:153
			assertMatch(t, true, "fo", "!(f)", win)            // extglobs-temp.js:154
			assertMatch(t, true, "foo", "!(f)", win)           // extglobs-temp.js:155
			assertMatch(t, false, "foo/bar", "!(f)", win)      // extglobs-temp.js:156
			assertMatch(t, true, "foobar", "!(f)", win)        // extglobs-temp.js:157
			assertMatch(t, true, "foot", "!(f)", win)          // extglobs-temp.js:158
			assertMatch(t, true, "foox", "!(f)", win)          // extglobs-temp.js:159
			assertMatch(t, true, "o", "!(f)", win)             // extglobs-temp.js:160
			assertMatch(t, true, "of", "!(f)", win)            // extglobs-temp.js:161
			assertMatch(t, true, "ooo", "!(f)", win)           // extglobs-temp.js:162
			assertMatch(t, true, "ox", "!(f)", win)            // extglobs-temp.js:163
			assertMatch(t, true, "x", "!(f)", win)             // extglobs-temp.js:164
			assertMatch(t, true, "xx", "!(f)", win)            // extglobs-temp.js:165

			// !(f) — second block (duplicate in JS source)
			assertMatch(t, true, "bar", "!(f)", win)           // extglobs-temp.js:167
			assertMatch(t, false, "f", "!(f)", win)            // extglobs-temp.js:168
			assertMatch(t, true, "fa", "!(f)", win)            // extglobs-temp.js:169
			assertMatch(t, true, "fb", "!(f)", win)            // extglobs-temp.js:170
			assertMatch(t, true, "ff", "!(f)", win)            // extglobs-temp.js:171
			assertMatch(t, true, "fff", "!(f)", win)           // extglobs-temp.js:172
			assertMatch(t, true, "fo", "!(f)", win)            // extglobs-temp.js:173
			assertMatch(t, true, "foo", "!(f)", win)           // extglobs-temp.js:174
			assertMatch(t, false, "foo/bar", "!(f)", win)      // extglobs-temp.js:175
			assertMatch(t, true, "foobar", "!(f)", win)        // extglobs-temp.js:176
			assertMatch(t, true, "foot", "!(f)", win)          // extglobs-temp.js:177
			assertMatch(t, true, "foox", "!(f)", win)          // extglobs-temp.js:178
			assertMatch(t, true, "o", "!(f)", win)             // extglobs-temp.js:179
			assertMatch(t, true, "of", "!(f)", win)            // extglobs-temp.js:180
			assertMatch(t, true, "ooo", "!(f)", win)           // extglobs-temp.js:181
			assertMatch(t, true, "ox", "!(f)", win)            // extglobs-temp.js:182
			assertMatch(t, true, "x", "!(f)", win)             // extglobs-temp.js:183
			assertMatch(t, true, "xx", "!(f)", win)            // extglobs-temp.js:184

			// !(foo) — second block (duplicate in JS source)
			assertMatch(t, true, "bar", "!(foo)", win)         // extglobs-temp.js:186
			assertMatch(t, true, "f", "!(foo)", win)           // extglobs-temp.js:187
			assertMatch(t, true, "fa", "!(foo)", win)          // extglobs-temp.js:188
			assertMatch(t, true, "fb", "!(foo)", win)          // extglobs-temp.js:189
			assertMatch(t, true, "ff", "!(foo)", win)          // extglobs-temp.js:190
			assertMatch(t, true, "fff", "!(foo)", win)         // extglobs-temp.js:191
			assertMatch(t, true, "fo", "!(foo)", win)          // extglobs-temp.js:192
			assertMatch(t, false, "foo", "!(foo)", win)        // extglobs-temp.js:193
			assertMatch(t, false, "foo/bar", "!(foo)", win)    // extglobs-temp.js:194
			assertMatch(t, true, "foobar", "!(foo)", win)      // extglobs-temp.js:195
			assertMatch(t, true, "foot", "!(foo)", win)        // extglobs-temp.js:196
			assertMatch(t, true, "foox", "!(foo)", win)        // extglobs-temp.js:197
			assertMatch(t, true, "o", "!(foo)", win)           // extglobs-temp.js:198
			assertMatch(t, true, "of", "!(foo)", win)          // extglobs-temp.js:199
			assertMatch(t, true, "ooo", "!(foo)", win)         // extglobs-temp.js:200
			assertMatch(t, true, "ox", "!(foo)", win)          // extglobs-temp.js:201
			assertMatch(t, true, "x", "!(foo)", win)           // extglobs-temp.js:202
			assertMatch(t, true, "xx", "!(foo)", win)          // extglobs-temp.js:203

			// !(foo)*
			assertMatch(t, true, "bar", "!(foo)*", win)        // extglobs-temp.js:205
			assertMatch(t, true, "f", "!(foo)*", win)          // extglobs-temp.js:206
			assertMatch(t, true, "fa", "!(foo)*", win)         // extglobs-temp.js:207
			assertMatch(t, true, "fb", "!(foo)*", win)         // extglobs-temp.js:208
			assertMatch(t, true, "ff", "!(foo)*", win)         // extglobs-temp.js:209
			assertMatch(t, true, "fff", "!(foo)*", win)        // extglobs-temp.js:210
			assertMatch(t, true, "fo", "!(foo)*", win)         // extglobs-temp.js:211
			assertMatch(t, false, "foo", "!(foo)*", win)       // extglobs-temp.js:212
			assertMatch(t, false, "foo/bar", "!(foo)*", win)   // extglobs-temp.js:213
			assertMatch(t, false, "foobar", "!(foo)*", win)    // extglobs-temp.js:214
			assertMatch(t, false, "foot", "!(foo)*", win)      // extglobs-temp.js:215
			assertMatch(t, false, "foox", "!(foo)*", win)      // extglobs-temp.js:216
			assertMatch(t, true, "o", "!(foo)*", win)          // extglobs-temp.js:217
			assertMatch(t, true, "of", "!(foo)*", win)         // extglobs-temp.js:218
			assertMatch(t, true, "ooo", "!(foo)*", win)        // extglobs-temp.js:219
			assertMatch(t, true, "ox", "!(foo)*", win)         // extglobs-temp.js:220
			assertMatch(t, true, "x", "!(foo)*", win)          // extglobs-temp.js:221
			assertMatch(t, true, "xx", "!(foo)*", win)         // extglobs-temp.js:222

			// !(x)
			assertMatch(t, true, "bar", "!(x)", win)           // extglobs-temp.js:224
			assertMatch(t, true, "f", "!(x)", win)             // extglobs-temp.js:225
			assertMatch(t, true, "fa", "!(x)", win)            // extglobs-temp.js:226
			assertMatch(t, true, "fb", "!(x)", win)            // extglobs-temp.js:227
			assertMatch(t, true, "ff", "!(x)", win)            // extglobs-temp.js:228
			assertMatch(t, true, "fff", "!(x)", win)           // extglobs-temp.js:229
			assertMatch(t, true, "fo", "!(x)", win)            // extglobs-temp.js:230
			assertMatch(t, true, "foo", "!(x)", win)           // extglobs-temp.js:231
			assertMatch(t, false, "foo/bar", "!(x)", win)      // extglobs-temp.js:232
			assertMatch(t, true, "foobar", "!(x)", win)        // extglobs-temp.js:233
			assertMatch(t, true, "foot", "!(x)", win)          // extglobs-temp.js:234
			assertMatch(t, true, "foox", "!(x)", win)          // extglobs-temp.js:235
			assertMatch(t, true, "o", "!(x)", win)             // extglobs-temp.js:236
			assertMatch(t, true, "of", "!(x)", win)            // extglobs-temp.js:237
			assertMatch(t, true, "ooo", "!(x)", win)           // extglobs-temp.js:238
			assertMatch(t, true, "ox", "!(x)", win)            // extglobs-temp.js:239
			assertMatch(t, false, "x", "!(x)", win)            // extglobs-temp.js:240
			assertMatch(t, true, "xx", "!(x)", win)            // extglobs-temp.js:241

			// !(x)*
			assertMatch(t, true, "bar", "!(x)*", win)          // extglobs-temp.js:243
			assertMatch(t, true, "f", "!(x)*", win)            // extglobs-temp.js:244
			assertMatch(t, true, "fa", "!(x)*", win)           // extglobs-temp.js:245
			assertMatch(t, true, "fb", "!(x)*", win)           // extglobs-temp.js:246
			assertMatch(t, true, "ff", "!(x)*", win)           // extglobs-temp.js:247
			assertMatch(t, true, "fff", "!(x)*", win)          // extglobs-temp.js:248
			assertMatch(t, true, "fo", "!(x)*", win)           // extglobs-temp.js:249
			assertMatch(t, true, "foo", "!(x)*", win)          // extglobs-temp.js:250
			assertMatch(t, false, "foo/bar", "!(x)*", win)     // extglobs-temp.js:251
			assertMatch(t, true, "foobar", "!(x)*", win)       // extglobs-temp.js:252
			assertMatch(t, true, "foot", "!(x)*", win)         // extglobs-temp.js:253
			assertMatch(t, true, "foox", "!(x)*", win)         // extglobs-temp.js:254
			assertMatch(t, true, "o", "!(x)*", win)            // extglobs-temp.js:255
			assertMatch(t, true, "of", "!(x)*", win)           // extglobs-temp.js:256
			assertMatch(t, true, "ooo", "!(x)*", win)          // extglobs-temp.js:257
			assertMatch(t, true, "ox", "!(x)*", win)           // extglobs-temp.js:258
			assertMatch(t, false, "x", "!(x)*", win)           // extglobs-temp.js:259
			assertMatch(t, false, "xx", "!(x)*", win)          // extglobs-temp.js:260

			// *(!(f))
			assertMatch(t, true, "bar", "*(!(f))", win)        // extglobs-temp.js:262
			assertMatch(t, false, "f", "*(!(f))", win)         // extglobs-temp.js:263
			assertMatch(t, true, "fa", "*(!(f))", win)         // extglobs-temp.js:264
			assertMatch(t, true, "fb", "*(!(f))", win)         // extglobs-temp.js:265
			assertMatch(t, true, "ff", "*(!(f))", win)         // extglobs-temp.js:266
			assertMatch(t, true, "fff", "*(!(f))", win)        // extglobs-temp.js:267
			assertMatch(t, true, "fo", "*(!(f))", win)         // extglobs-temp.js:268
			assertMatch(t, true, "foo", "*(!(f))", win)        // extglobs-temp.js:269
			assertMatch(t, false, "foo/bar", "*(!(f))", win)   // extglobs-temp.js:270
			assertMatch(t, true, "foobar", "*(!(f))", win)     // extglobs-temp.js:271
			assertMatch(t, true, "foot", "*(!(f))", win)       // extglobs-temp.js:272
			assertMatch(t, true, "foox", "*(!(f))", win)       // extglobs-temp.js:273
			assertMatch(t, true, "o", "*(!(f))", win)          // extglobs-temp.js:274
			assertMatch(t, true, "of", "*(!(f))", win)         // extglobs-temp.js:275
			assertMatch(t, true, "ooo", "*(!(f))", win)        // extglobs-temp.js:276
			assertMatch(t, true, "ox", "*(!(f))", win)         // extglobs-temp.js:277
			assertMatch(t, true, "x", "*(!(f))", win)          // extglobs-temp.js:278
			assertMatch(t, true, "xx", "*(!(f))", win)         // extglobs-temp.js:279

			// *((foo))
			assertMatch(t, false, "bar", "*((foo))", win)      // extglobs-temp.js:281
			assertMatch(t, false, "f", "*((foo))", win)        // extglobs-temp.js:282
			assertMatch(t, false, "fa", "*((foo))", win)       // extglobs-temp.js:283
			assertMatch(t, false, "fb", "*((foo))", win)       // extglobs-temp.js:284
			assertMatch(t, false, "ff", "*((foo))", win)       // extglobs-temp.js:285
			assertMatch(t, false, "fff", "*((foo))", win)      // extglobs-temp.js:286
			assertMatch(t, false, "fo", "*((foo))", win)       // extglobs-temp.js:287
			assertMatch(t, true, "foo", "*((foo))", win)       // extglobs-temp.js:288
			assertMatch(t, false, "foo/bar", "*((foo))", win)  // extglobs-temp.js:289
			assertMatch(t, false, "foobar", "*((foo))", win)   // extglobs-temp.js:290
			assertMatch(t, false, "foot", "*((foo))", win)     // extglobs-temp.js:291
			assertMatch(t, false, "foox", "*((foo))", win)     // extglobs-temp.js:292
			assertMatch(t, false, "o", "*((foo))", win)        // extglobs-temp.js:293
			assertMatch(t, false, "of", "*((foo))", win)       // extglobs-temp.js:294
			assertMatch(t, false, "ooo", "*((foo))", win)      // extglobs-temp.js:295
			assertMatch(t, false, "ox", "*((foo))", win)       // extglobs-temp.js:296
			assertMatch(t, false, "x", "*((foo))", win)        // extglobs-temp.js:297
			assertMatch(t, false, "xx", "*((foo))", win)       // extglobs-temp.js:298

			// +(!(f))
			assertMatch(t, true, "bar", "+(!(f))", win)        // extglobs-temp.js:300
			assertMatch(t, false, "f", "+(!(f))", win)         // extglobs-temp.js:301
			assertMatch(t, true, "fa", "+(!(f))", win)         // extglobs-temp.js:302
			assertMatch(t, true, "fb", "+(!(f))", win)         // extglobs-temp.js:303
			assertMatch(t, true, "ff", "+(!(f))", win)         // extglobs-temp.js:304
			assertMatch(t, true, "fff", "+(!(f))", win)        // extglobs-temp.js:305
			assertMatch(t, true, "fo", "+(!(f))", win)         // extglobs-temp.js:306
			assertMatch(t, true, "foo", "+(!(f))", win)        // extglobs-temp.js:307
			assertMatch(t, false, "foo/bar", "+(!(f))", win)   // extglobs-temp.js:308
			assertMatch(t, true, "foobar", "+(!(f))", win)     // extglobs-temp.js:309
			assertMatch(t, true, "foot", "+(!(f))", win)       // extglobs-temp.js:310
			assertMatch(t, true, "foox", "+(!(f))", win)       // extglobs-temp.js:311
			assertMatch(t, true, "o", "+(!(f))", win)          // extglobs-temp.js:312
			assertMatch(t, true, "of", "+(!(f))", win)         // extglobs-temp.js:313
			assertMatch(t, true, "ooo", "+(!(f))", win)        // extglobs-temp.js:314
			assertMatch(t, true, "ox", "+(!(f))", win)         // extglobs-temp.js:315
			assertMatch(t, true, "x", "+(!(f))", win)          // extglobs-temp.js:316
			assertMatch(t, true, "xx", "+(!(f))", win)         // extglobs-temp.js:317

			// @(!(z*)|*x)
			assertMatch(t, true, "bar", "@(!(z*)|*x)", win)           // extglobs-temp.js:319
			assertMatch(t, true, "f", "@(!(z*)|*x)", win)             // extglobs-temp.js:320
			assertMatch(t, true, "fa", "@(!(z*)|*x)", win)            // extglobs-temp.js:321
			assertMatch(t, true, "fb", "@(!(z*)|*x)", win)            // extglobs-temp.js:322
			assertMatch(t, true, "ff", "@(!(z*)|*x)", win)            // extglobs-temp.js:323
			assertMatch(t, true, "fff", "@(!(z*)|*x)", win)           // extglobs-temp.js:324
			assertMatch(t, true, "fo", "@(!(z*)|*x)", win)            // extglobs-temp.js:325
			assertMatch(t, true, "foo", "@(!(z*)|*x)", win)           // extglobs-temp.js:326
			assertMatch(t, true, "foo/bar", "@(!(z*/*)|*x)", win)     // extglobs-temp.js:327
			assertMatch(t, false, "foo/bar", "@(!(z*)|*x)", win)      // extglobs-temp.js:328
			assertMatch(t, true, "foobar", "@(!(z*)|*x)", win)        // extglobs-temp.js:329
			assertMatch(t, true, "foot", "@(!(z*)|*x)", win)          // extglobs-temp.js:330
			assertMatch(t, true, "foox", "@(!(z*)|*x)", win)          // extglobs-temp.js:331
			assertMatch(t, true, "o", "@(!(z*)|*x)", win)             // extglobs-temp.js:332
			assertMatch(t, true, "of", "@(!(z*)|*x)", win)            // extglobs-temp.js:333
			assertMatch(t, true, "ooo", "@(!(z*)|*x)", win)           // extglobs-temp.js:334
			assertMatch(t, true, "ox", "@(!(z*)|*x)", win)            // extglobs-temp.js:335
			assertMatch(t, true, "x", "@(!(z*)|*x)", win)             // extglobs-temp.js:336
			assertMatch(t, true, "xx", "@(!(z*)|*x)", win)            // extglobs-temp.js:337

			// foo/!(foo)
			assertMatch(t, false, "bar", "foo/!(foo)", win)    // extglobs-temp.js:339
			assertMatch(t, false, "f", "foo/!(foo)", win)      // extglobs-temp.js:340
			assertMatch(t, false, "fa", "foo/!(foo)", win)     // extglobs-temp.js:341
			assertMatch(t, false, "fb", "foo/!(foo)", win)     // extglobs-temp.js:342
			assertMatch(t, false, "ff", "foo/!(foo)", win)     // extglobs-temp.js:343
			assertMatch(t, false, "fff", "foo/!(foo)", win)    // extglobs-temp.js:344
			assertMatch(t, false, "fo", "foo/!(foo)", win)     // extglobs-temp.js:345
			assertMatch(t, false, "foo", "foo/!(foo)", win)    // extglobs-temp.js:346
			assertMatch(t, true, "foo/bar", "foo/!(foo)", win) // extglobs-temp.js:347
			assertMatch(t, false, "foobar", "foo/!(foo)", win) // extglobs-temp.js:348
			assertMatch(t, false, "foot", "foo/!(foo)", win)   // extglobs-temp.js:349
			assertMatch(t, false, "foox", "foo/!(foo)", win)   // extglobs-temp.js:350
			assertMatch(t, false, "o", "foo/!(foo)", win)      // extglobs-temp.js:351
			assertMatch(t, false, "of", "foo/!(foo)", win)     // extglobs-temp.js:352
			assertMatch(t, false, "ooo", "foo/!(foo)", win)    // extglobs-temp.js:353
			assertMatch(t, false, "ox", "foo/!(foo)", win)     // extglobs-temp.js:354
			assertMatch(t, false, "x", "foo/!(foo)", win)      // extglobs-temp.js:355
			assertMatch(t, false, "xx", "foo/!(foo)", win)     // extglobs-temp.js:356

			// (foo)bb
			assertMatch(t, false, "ffffffo", "(foo)bb", win)                  // extglobs-temp.js:358
			assertMatch(t, false, "fffooofoooooffoofffooofff", "(foo)bb", win) // extglobs-temp.js:359
			assertMatch(t, false, "ffo", "(foo)bb", win)                      // extglobs-temp.js:360
			assertMatch(t, false, "fofo", "(foo)bb", win)                     // extglobs-temp.js:361
			assertMatch(t, false, "fofoofoofofoo", "(foo)bb", win)            // extglobs-temp.js:362
			assertMatch(t, false, "foo", "(foo)bb", win)                      // extglobs-temp.js:363
			assertMatch(t, false, "foob", "(foo)bb", win)                     // extglobs-temp.js:364
			assertMatch(t, true, "foobb", "(foo)bb", win)                     // extglobs-temp.js:365
			assertMatch(t, false, "foofoofo", "(foo)bb", win)                 // extglobs-temp.js:366
			assertMatch(t, false, "fooofoofofooo", "(foo)bb", win)            // extglobs-temp.js:367
			assertMatch(t, false, "foooofo", "(foo)bb", win)                  // extglobs-temp.js:368
			assertMatch(t, false, "foooofof", "(foo)bb", win)                 // extglobs-temp.js:369
			assertMatch(t, false, "foooofofx", "(foo)bb", win)                // extglobs-temp.js:370
			assertMatch(t, false, "foooxfooxfoxfooox", "(foo)bb", win)        // extglobs-temp.js:371
			assertMatch(t, false, "foooxfooxfxfooox", "(foo)bb", win)         // extglobs-temp.js:372
			assertMatch(t, false, "foooxfooxofoxfooox", "(foo)bb", win)       // extglobs-temp.js:373
			assertMatch(t, false, "foot", "(foo)bb", win)                     // extglobs-temp.js:374
			assertMatch(t, false, "foox", "(foo)bb", win)                     // extglobs-temp.js:375
			assertMatch(t, false, "ofoofo", "(foo)bb", win)                   // extglobs-temp.js:376
			assertMatch(t, false, "ofooofoofofooo", "(foo)bb", win)           // extglobs-temp.js:377
			assertMatch(t, false, "ofoooxoofxo", "(foo)bb", win)              // extglobs-temp.js:378
			assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "(foo)bb", win)   // extglobs-temp.js:379
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "(foo)bb", win) // extglobs-temp.js:380
			assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "(foo)bb", win)  // extglobs-temp.js:381
			assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "(foo)bb", win) // extglobs-temp.js:382
			assertMatch(t, false, "ofxoofxo", "(foo)bb", win)                 // extglobs-temp.js:383
			assertMatch(t, false, "oofooofo", "(foo)bb", win)                 // extglobs-temp.js:384
			assertMatch(t, false, "ooo", "(foo)bb", win)                      // extglobs-temp.js:385
			assertMatch(t, false, "oxfoxfox", "(foo)bb", win)                 // extglobs-temp.js:386
			assertMatch(t, false, "oxfoxoxfox", "(foo)bb", win)               // extglobs-temp.js:387
			assertMatch(t, false, "xfoooofof", "(foo)bb", win)                // extglobs-temp.js:388

			// *(*(f)*(o))
			assertMatch(t, true, "ffffffo", "*(*(f)*(o))", win)                  // extglobs-temp.js:390
			assertMatch(t, true, "fffooofoooooffoofffooofff", "*(*(f)*(o))", win) // extglobs-temp.js:391
			assertMatch(t, true, "ffo", "*(*(f)*(o))", win)                      // extglobs-temp.js:392
			assertMatch(t, true, "fofo", "*(*(f)*(o))", win)                     // extglobs-temp.js:393
			assertMatch(t, true, "fofoofoofofoo", "*(*(f)*(o))", win)            // extglobs-temp.js:394
			assertMatch(t, true, "foo", "*(*(f)*(o))", win)                      // extglobs-temp.js:395
			assertMatch(t, false, "foob", "*(*(f)*(o))", win)                    // extglobs-temp.js:396
			assertMatch(t, false, "foobb", "*(*(f)*(o))", win)                   // extglobs-temp.js:397
			assertMatch(t, true, "foofoofo", "*(*(f)*(o))", win)                 // extglobs-temp.js:398
			assertMatch(t, true, "fooofoofofooo", "*(*(f)*(o))", win)            // extglobs-temp.js:399
			assertMatch(t, true, "foooofo", "*(*(f)*(o))", win)                  // extglobs-temp.js:400
			assertMatch(t, true, "foooofof", "*(*(f)*(o))", win)                 // extglobs-temp.js:401
			assertMatch(t, false, "foooofofx", "*(*(f)*(o))", win)               // extglobs-temp.js:402
			assertMatch(t, false, "foooxfooxfoxfooox", "*(*(f)*(o))", win)       // extglobs-temp.js:403
			assertMatch(t, false, "foooxfooxfxfooox", "*(*(f)*(o))", win)        // extglobs-temp.js:404
			assertMatch(t, false, "foooxfooxofoxfooox", "*(*(f)*(o))", win)      // extglobs-temp.js:405
			assertMatch(t, false, "foot", "*(*(f)*(o))", win)                    // extglobs-temp.js:406
			assertMatch(t, false, "foox", "*(*(f)*(o))", win)                    // extglobs-temp.js:407
			assertMatch(t, true, "ofoofo", "*(*(f)*(o))", win)                   // extglobs-temp.js:408
			assertMatch(t, true, "ofooofoofofooo", "*(*(f)*(o))", win)           // extglobs-temp.js:409
			assertMatch(t, false, "ofoooxoofxo", "*(*(f)*(o))", win)             // extglobs-temp.js:410
			assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(*(f)*(o))", win)  // extglobs-temp.js:411
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(*(f)*(o))", win) // extglobs-temp.js:412
			assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(*(f)*(o))", win) // extglobs-temp.js:413
			assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(*(f)*(o))", win) // extglobs-temp.js:414
			assertMatch(t, false, "ofxoofxo", "*(*(f)*(o))", win)                // extglobs-temp.js:415
			assertMatch(t, true, "oofooofo", "*(*(f)*(o))", win)                 // extglobs-temp.js:416
			assertMatch(t, true, "ooo", "*(*(f)*(o))", win)                      // extglobs-temp.js:417
			assertMatch(t, false, "oxfoxfox", "*(*(f)*(o))", win)                // extglobs-temp.js:418
			assertMatch(t, false, "oxfoxoxfox", "*(*(f)*(o))", win)              // extglobs-temp.js:419
			assertMatch(t, false, "xfoooofof", "*(*(f)*(o))", win)               // extglobs-temp.js:420

			// *(*(of*(o)x)o)
			assertMatch(t, false, "ffffffo", "*(*(of*(o)x)o)", win)                  // extglobs-temp.js:422
			assertMatch(t, false, "fffooofoooooffoofffooofff", "*(*(of*(o)x)o)", win) // extglobs-temp.js:423
			assertMatch(t, false, "ffo", "*(*(of*(o)x)o)", win)                      // extglobs-temp.js:424
			assertMatch(t, false, "fofo", "*(*(of*(o)x)o)", win)                     // extglobs-temp.js:425
			assertMatch(t, false, "fofoofoofofoo", "*(*(of*(o)x)o)", win)            // extglobs-temp.js:426
			assertMatch(t, false, "foo", "*(*(of*(o)x)o)", win)                      // extglobs-temp.js:427
			assertMatch(t, false, "foob", "*(*(of*(o)x)o)", win)                     // extglobs-temp.js:428
			assertMatch(t, false, "foobb", "*(*(of*(o)x)o)", win)                    // extglobs-temp.js:429
			assertMatch(t, false, "foofoofo", "*(*(of*(o)x)o)", win)                 // extglobs-temp.js:430
			assertMatch(t, false, "fooofoofofooo", "*(*(of*(o)x)o)", win)            // extglobs-temp.js:431
			assertMatch(t, false, "foooofo", "*(*(of*(o)x)o)", win)                  // extglobs-temp.js:432
			assertMatch(t, false, "foooofof", "*(*(of*(o)x)o)", win)                 // extglobs-temp.js:433
			assertMatch(t, false, "foooofofx", "*(*(of*(o)x)o)", win)                // extglobs-temp.js:434
			assertMatch(t, false, "foooxfooxfoxfooox", "*(*(of*(o)x)o)", win)        // extglobs-temp.js:435
			assertMatch(t, false, "foooxfooxfxfooox", "*(*(of*(o)x)o)", win)         // extglobs-temp.js:436
			assertMatch(t, false, "foooxfooxofoxfooox", "*(*(of*(o)x)o)", win)       // extglobs-temp.js:437
			assertMatch(t, false, "foot", "*(*(of*(o)x)o)", win)                     // extglobs-temp.js:438
			assertMatch(t, false, "foox", "*(*(of*(o)x)o)", win)                     // extglobs-temp.js:439
			assertMatch(t, false, "ofoofo", "*(*(of*(o)x)o)", win)                   // extglobs-temp.js:440
			assertMatch(t, false, "ofooofoofofooo", "*(*(of*(o)x)o)", win)           // extglobs-temp.js:441
			assertMatch(t, true, "ofoooxoofxo", "*(*(of*(o)x)o)", win)               // extglobs-temp.js:442
			assertMatch(t, true, "ofoooxoofxoofoooxoofxo", "*(*(of*(o)x)o)", win)    // extglobs-temp.js:443
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(*(of*(o)x)o)", win) // extglobs-temp.js:444
			assertMatch(t, true, "ofoooxoofxoofoooxoofxoo", "*(*(of*(o)x)o)", win)   // extglobs-temp.js:445
			assertMatch(t, true, "ofoooxoofxoofoooxoofxooofxofxo", "*(*(of*(o)x)o)", win) // extglobs-temp.js:446
			assertMatch(t, true, "ofxoofxo", "*(*(of*(o)x)o)", win)                  // extglobs-temp.js:447
			assertMatch(t, false, "oofooofo", "*(*(of*(o)x)o)", win)                 // extglobs-temp.js:448
			assertMatch(t, true, "ooo", "*(*(of*(o)x)o)", win)                       // extglobs-temp.js:449
			assertMatch(t, false, "oxfoxfox", "*(*(of*(o)x)o)", win)                 // extglobs-temp.js:450
			assertMatch(t, false, "oxfoxoxfox", "*(*(of*(o)x)o)", win)               // extglobs-temp.js:451
			assertMatch(t, false, "xfoooofof", "*(*(of*(o)x)o)", win)                // extglobs-temp.js:452

			// *(f*(o))
			assertMatch(t, true, "ffffffo", "*(f*(o))", win)                  // extglobs-temp.js:454
			assertMatch(t, true, "fffooofoooooffoofffooofff", "*(f*(o))", win) // extglobs-temp.js:455
			assertMatch(t, true, "ffo", "*(f*(o))", win)                      // extglobs-temp.js:456
			assertMatch(t, true, "fofo", "*(f*(o))", win)                     // extglobs-temp.js:457
			assertMatch(t, true, "fofoofoofofoo", "*(f*(o))", win)            // extglobs-temp.js:458
			assertMatch(t, true, "foo", "*(f*(o))", win)                      // extglobs-temp.js:459
			assertMatch(t, false, "foob", "*(f*(o))", win)                    // extglobs-temp.js:460
			assertMatch(t, false, "foobb", "*(f*(o))", win)                   // extglobs-temp.js:461
			assertMatch(t, true, "foofoofo", "*(f*(o))", win)                 // extglobs-temp.js:462
			assertMatch(t, true, "fooofoofofooo", "*(f*(o))", win)            // extglobs-temp.js:463
			assertMatch(t, true, "foooofo", "*(f*(o))", win)                  // extglobs-temp.js:464
			assertMatch(t, true, "foooofof", "*(f*(o))", win)                 // extglobs-temp.js:465
			assertMatch(t, false, "foooofofx", "*(f*(o))", win)               // extglobs-temp.js:466
			assertMatch(t, false, "foooxfooxfoxfooox", "*(f*(o))", win)       // extglobs-temp.js:467
			assertMatch(t, false, "foooxfooxfxfooox", "*(f*(o))", win)        // extglobs-temp.js:468
			assertMatch(t, false, "foooxfooxofoxfooox", "*(f*(o))", win)      // extglobs-temp.js:469
			assertMatch(t, false, "foot", "*(f*(o))", win)                    // extglobs-temp.js:470
			assertMatch(t, false, "foox", "*(f*(o))", win)                    // extglobs-temp.js:471
			assertMatch(t, false, "ofoofo", "*(f*(o))", win)                  // extglobs-temp.js:472
			assertMatch(t, false, "ofooofoofofooo", "*(f*(o))", win)          // extglobs-temp.js:473
			assertMatch(t, false, "ofoooxoofxo", "*(f*(o))", win)             // extglobs-temp.js:474
			assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(f*(o))", win)  // extglobs-temp.js:475
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(f*(o))", win) // extglobs-temp.js:476
			assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(f*(o))", win) // extglobs-temp.js:477
			assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(f*(o))", win) // extglobs-temp.js:478
			assertMatch(t, false, "ofxoofxo", "*(f*(o))", win)                // extglobs-temp.js:479
			assertMatch(t, false, "oofooofo", "*(f*(o))", win)                // extglobs-temp.js:480
			assertMatch(t, false, "ooo", "*(f*(o))", win)                     // extglobs-temp.js:481
			assertMatch(t, false, "oxfoxfox", "*(f*(o))", win)                // extglobs-temp.js:482
			assertMatch(t, false, "oxfoxoxfox", "*(f*(o))", win)              // extglobs-temp.js:483
			assertMatch(t, false, "xfoooofof", "*(f*(o))", win)               // extglobs-temp.js:484

			// *(f*(o)x)
			assertMatch(t, false, "ffffffo", "*(f*(o)x)", win)                  // extglobs-temp.js:486
			assertMatch(t, false, "fffooofoooooffoofffooofff", "*(f*(o)x)", win) // extglobs-temp.js:487
			assertMatch(t, false, "ffo", "*(f*(o)x)", win)                      // extglobs-temp.js:488
			assertMatch(t, false, "fofo", "*(f*(o)x)", win)                     // extglobs-temp.js:489
			assertMatch(t, false, "fofoofoofofoo", "*(f*(o)x)", win)            // extglobs-temp.js:490
			assertMatch(t, false, "foo", "*(f*(o)x)", win)                      // extglobs-temp.js:491
			assertMatch(t, false, "foob", "*(f*(o)x)", win)                     // extglobs-temp.js:492
			assertMatch(t, false, "foobb", "*(f*(o)x)", win)                    // extglobs-temp.js:493
			assertMatch(t, false, "foofoofo", "*(f*(o)x)", win)                 // extglobs-temp.js:494
			assertMatch(t, false, "fooofoofofooo", "*(f*(o)x)", win)            // extglobs-temp.js:495
			assertMatch(t, false, "foooofo", "*(f*(o)x)", win)                  // extglobs-temp.js:496
			assertMatch(t, false, "foooofof", "*(f*(o)x)", win)                 // extglobs-temp.js:497
			assertMatch(t, false, "foooofofx", "*(f*(o)x)", win)                // extglobs-temp.js:498
			assertMatch(t, true, "foooxfooxfoxfooox", "*(f*(o)x)", win)         // extglobs-temp.js:499
			assertMatch(t, true, "foooxfooxfxfooox", "*(f*(o)x)", win)          // extglobs-temp.js:500
			assertMatch(t, false, "foooxfooxofoxfooox", "*(f*(o)x)", win)       // extglobs-temp.js:501
			assertMatch(t, false, "foot", "*(f*(o)x)", win)                     // extglobs-temp.js:502
			assertMatch(t, true, "foox", "*(f*(o)x)", win)                      // extglobs-temp.js:503
			assertMatch(t, false, "ofoofo", "*(f*(o)x)", win)                   // extglobs-temp.js:504
			assertMatch(t, false, "ofooofoofofooo", "*(f*(o)x)", win)           // extglobs-temp.js:505
			assertMatch(t, false, "ofoooxoofxo", "*(f*(o)x)", win)              // extglobs-temp.js:506
			assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(f*(o)x)", win)   // extglobs-temp.js:507
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(f*(o)x)", win) // extglobs-temp.js:508
			assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(f*(o)x)", win)  // extglobs-temp.js:509
			assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(f*(o)x)", win) // extglobs-temp.js:510
			assertMatch(t, false, "ofxoofxo", "*(f*(o)x)", win)                 // extglobs-temp.js:511
			assertMatch(t, false, "oofooofo", "*(f*(o)x)", win)                 // extglobs-temp.js:512
			assertMatch(t, false, "ooo", "*(f*(o)x)", win)                      // extglobs-temp.js:513
			assertMatch(t, false, "oxfoxfox", "*(f*(o)x)", win)                 // extglobs-temp.js:514
			assertMatch(t, false, "oxfoxoxfox", "*(f*(o)x)", win)               // extglobs-temp.js:515
			assertMatch(t, false, "xfoooofof", "*(f*(o)x)", win)                // extglobs-temp.js:516

			// *(f+(o))
			assertMatch(t, false, "ffffffo", "*(f+(o))", win)                  // extglobs-temp.js:518
			assertMatch(t, false, "fffooofoooooffoofffooofff", "*(f+(o))", win) // extglobs-temp.js:519
			assertMatch(t, false, "ffo", "*(f+(o))", win)                      // extglobs-temp.js:520
			assertMatch(t, true, "fofo", "*(f+(o))", win)                      // extglobs-temp.js:521
			assertMatch(t, true, "fofoofoofofoo", "*(f+(o))", win)             // extglobs-temp.js:522
			assertMatch(t, true, "foo", "*(f+(o))", win)                       // extglobs-temp.js:523
			assertMatch(t, false, "foob", "*(f+(o))", win)                     // extglobs-temp.js:524
			assertMatch(t, false, "foobb", "*(f+(o))", win)                    // extglobs-temp.js:525
			assertMatch(t, true, "foofoofo", "*(f+(o))", win)                  // extglobs-temp.js:526
			assertMatch(t, true, "fooofoofofooo", "*(f+(o))", win)             // extglobs-temp.js:527
			assertMatch(t, true, "foooofo", "*(f+(o))", win)                   // extglobs-temp.js:528
			assertMatch(t, false, "foooofof", "*(f+(o))", win)                 // extglobs-temp.js:529
			assertMatch(t, false, "foooofofx", "*(f+(o))", win)                // extglobs-temp.js:530
			assertMatch(t, false, "foooxfooxfoxfooox", "*(f+(o))", win)        // extglobs-temp.js:531
			assertMatch(t, false, "foooxfooxfxfooox", "*(f+(o))", win)         // extglobs-temp.js:532
			assertMatch(t, false, "foooxfooxofoxfooox", "*(f+(o))", win)       // extglobs-temp.js:533
			assertMatch(t, false, "foot", "*(f+(o))", win)                     // extglobs-temp.js:534
			assertMatch(t, false, "foox", "*(f+(o))", win)                     // extglobs-temp.js:535
			assertMatch(t, false, "ofoofo", "*(f+(o))", win)                   // extglobs-temp.js:536
			assertMatch(t, false, "ofooofoofofooo", "*(f+(o))", win)           // extglobs-temp.js:537
			assertMatch(t, false, "ofoooxoofxo", "*(f+(o))", win)              // extglobs-temp.js:538
			assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(f+(o))", win)   // extglobs-temp.js:539
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(f+(o))", win) // extglobs-temp.js:540
			assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(f+(o))", win)  // extglobs-temp.js:541
			assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(f+(o))", win) // extglobs-temp.js:542
			assertMatch(t, false, "ofxoofxo", "*(f+(o))", win)                 // extglobs-temp.js:543
			assertMatch(t, false, "oofooofo", "*(f+(o))", win)                 // extglobs-temp.js:544
			assertMatch(t, false, "ooo", "*(f+(o))", win)                      // extglobs-temp.js:545
			assertMatch(t, false, "oxfoxfox", "*(f+(o))", win)                 // extglobs-temp.js:546
			assertMatch(t, false, "oxfoxoxfox", "*(f+(o))", win)               // extglobs-temp.js:547
			assertMatch(t, false, "xfoooofof", "*(f+(o))", win)                // extglobs-temp.js:548

			// *(of+(o))
			assertMatch(t, false, "ffffffo", "*(of+(o))", win)                  // extglobs-temp.js:550
			assertMatch(t, false, "fffooofoooooffoofffooofff", "*(of+(o))", win) // extglobs-temp.js:551
			assertMatch(t, false, "ffo", "*(of+(o))", win)                      // extglobs-temp.js:552
			assertMatch(t, false, "fofo", "*(of+(o))", win)                     // extglobs-temp.js:553
			assertMatch(t, false, "fofoofoofofoo", "*(of+(o))", win)            // extglobs-temp.js:554
			assertMatch(t, false, "foo", "*(of+(o))", win)                      // extglobs-temp.js:555
			assertMatch(t, false, "foob", "*(of+(o))", win)                     // extglobs-temp.js:556
			assertMatch(t, false, "foobb", "*(of+(o))", win)                    // extglobs-temp.js:557
			assertMatch(t, false, "foofoofo", "*(of+(o))", win)                 // extglobs-temp.js:558
			assertMatch(t, false, "fooofoofofooo", "*(of+(o))", win)            // extglobs-temp.js:559
			assertMatch(t, false, "foooofo", "*(of+(o))", win)                  // extglobs-temp.js:560
			assertMatch(t, false, "foooofof", "*(of+(o))", win)                 // extglobs-temp.js:561
			assertMatch(t, false, "foooofofx", "*(of+(o))", win)                // extglobs-temp.js:562
			assertMatch(t, false, "foooxfooxfoxfooox", "*(of+(o))", win)        // extglobs-temp.js:563
			assertMatch(t, false, "foooxfooxfxfooox", "*(of+(o))", win)         // extglobs-temp.js:564
			assertMatch(t, false, "foooxfooxofoxfooox", "*(of+(o))", win)       // extglobs-temp.js:565
			assertMatch(t, false, "foot", "*(of+(o))", win)                     // extglobs-temp.js:566
			assertMatch(t, false, "foox", "*(of+(o))", win)                     // extglobs-temp.js:567
			assertMatch(t, true, "ofoofo", "*(of+(o))", win)                    // extglobs-temp.js:568
			assertMatch(t, false, "ofooofoofofooo", "*(of+(o))", win)           // extglobs-temp.js:569
			assertMatch(t, false, "ofoooxoofxo", "*(of+(o))", win)              // extglobs-temp.js:570
			assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(of+(o))", win)   // extglobs-temp.js:571
			assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(of+(o))", win) // extglobs-temp.js:572
			assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(of+(o))", win)  // extglobs-temp.js:573
			assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(of+(o))", win) // extglobs-temp.js:574
			assertMatch(t, false, "ofxoofxo", "*(of+(o))", win)                 // extglobs-temp.js:575
			assertMatch(t, false, "oofooofo", "*(of+(o))", win)                 // extglobs-temp.js:576
			assertMatch(t, false, "ooo", "*(of+(o))", win)                      // extglobs-temp.js:577
			assertMatch(t, false, "oxfoxfox", "*(of+(o))", win)                 // extglobs-temp.js:578
			assertMatch(t, false, "oxfoxoxfox", "*(of+(o))", win)               // extglobs-temp.js:579
			assertMatch(t, false, "xfoooofof", "*(of+(o))", win)                // extglobs-temp.js:580

			// *(of+(o)|f)
			assertMatch(t, false, "ffffffo", "*(of+(o)|f)", win)                  // extglobs-temp.js:582
			assertMatch(t, false, "fffooofoooooffoofffooofff", "*(of+(o)|f)", win) // extglobs-temp.js:583
			assertMatch(t, false, "ffo", "*(of+(o)|f)", win)                      // extglobs-temp.js:584
			assertMatch(t, true, "fofo", "*(of+(o)|f)", win)                      // extglobs-temp.js:585
			assertMatch(t, true, "fofoofoofofoo", "*(of+(o)|f)", win)             // extglobs-temp.js:586
			assertMatch(t, false, "foo", "*(of+(o)|f)", win)                      // extglobs-temp.js:587
			assertMatch(t, false, "foob", "*(of+(o)|f)", win)                     // extglobs-temp.js:588
			assertMatch(t, false, "foobb", "*(of+(o)|f)", win)                    // extglobs-temp.js:589
			assertMatch(t, false, "foofoofo", "*(of+(o)|f)", win)                 // extglobs-temp.js:590
			assertMatch(t, false, "fooofoofofooo", "*(of+(o)|f)", win)            // extglobs-temp.js:591
			assertMatch(t, false, "foooofo", "*(of+(o)|f)", win)                  // extglobs-temp.js:592
			assertMatch(t, false, "foooofof", "*(of+(o)|f)", win)                 // extglobs-temp.js:593
			assertMatch(t, false, "foooofofx", "*(of+(o)|f)", win)                // extglobs-temp.js:594
			assertMatch(t, false, "foooxfooxfoxfooox", "*(of+(o)|f)", win)        // extglobs-temp.js:595
			assertMatch(t, false, "foooxfooxfxfooox", "*(of+(o)|f)", win)         // extglobs-temp.js:596
			assertMatch(t, false, "foooxfooxofoxfooox", "*(of+(o)|f)", win)       // extglobs-temp.js:597
			assertMatch(t, false, "foot", "*(of+(o)|f)", win)                     // extglobs-temp.js:598
			assertMatch(t, false, "foox", "*(of+(o)|f)", win)                     // extglobs-temp.js:599
			assertMatch(t, true, "ofoofo", "*(of+(o)|f)", win)                    // extglobs-temp.js:600
		})
	})
}
