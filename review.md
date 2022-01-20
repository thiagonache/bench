* As a package name, what about just `bench`? I mean, you don't need to say `simple`, do you? All your packages should be simple! There's nothing stopping you calling your _CLI tool_ `simplebench` if you like: that doesn't also need to be the name of the Go package that implements it.

* Since test names should contain a verb (because they describe some desired behaviour), we might rename `TestRequestNonOK` to something like `TestNonOKStatusRecordedAsFailure`—it's not crucial, but this is a nice way to think about your test names. If you name your test functions well, you often don't need to say anything else in the failure message: the name says it all!

* You maybe now don't need to check `called` in this test; it can hardly pass unless the test server is called, can it? On the other hand, you might prefer to extract this out to some more modest test, that the `LoadGen` actually _makes_ any requests at all, regardless of how it handles their response status.

* Instead of `bench.LoadGen`, what about `bench.Tester`? It's always nice when your types or functions read like plain English.

* It doesn't seem like it would make sense to have a public API like `loadgen.AddToWG(1)`. And according to the test, it doesn't seem to have any user-facing behaviour. We don't call something like `loadgen.WGWait`, for example.

I think it's worth being clear in your mind about how your API deals with concurrency, isn't it? For example, you might keep it completely hidden, and simply expose a blocking entry point like `loadgen.Run()`. Or you might provide both concurrent and synchronous APIs, or take a context to allow cancellation, or concurrent only—the possibilities are varied! Design problems are much harder with concurrent programs than sequential ones.

* In `TestNewLoadGenDefault`, we check that calling `loadgen.GetHTTPUserAgent` returns the string we expect, but that's all. We don't test, for example, that the loadgen actually _sends_ that user agent string as part of requests—and we could do that, couldn't we? It doesn't seem that `GetHTTPUserAgent` would be useful for anything other than this test, anyway.

* We shouldn't modify `http.DefaultClient` at line 68. Indeed, there's no reason to refer to it at all. We can create an `http.Client` literal with the necessary timeout.

* The names `wantHTTPClient` and `gotHTTPClient` are too long, and since we repeat them four times in one case and three in the other, it's especially painful. Just `want` and `got` are ideal. Indeed, we test three different things in this test, so `want` and `got` would be of three different types, and that's a smell that they should really be three separate tests.

* I think it might end up being clearer to have one test about the requests config (1 by default, or whatever you set with an option), one about the user agent (the default by default, or whatever you set with an option), and one about the HTTP client (the default by default, or whatever you set with an option). That way, you expand two tests to three, but each one is more focused.

* In `TestURLParseValid`, we should report the unexpected error itself, rather than say simply 'error'. Because in the event that the test fails, it's because `NewLoadGen` is incorrectly reporting an error. How can we tell what's wrong unless we know what that error _is_?
