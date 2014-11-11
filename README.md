# monitor

HTTP monitoring tool.  `GET` an endpoint at some frequency, with some
concurrency, for some duration. See how it reacts.

# Demo

![demo-of-command](https://raw.githubusercontent.com/aybabtme/monitor/master/demo.gif)

## load testing

This tool is not meant as a load tester, although it can likely serve
like that.

It's really more meant to get a rough idea of the performance of an HTTP
endpoint. It's not fancy at all.

Checkout [`boom`][boom] or [`vegeta`][vegeta] if you want something more fancy in Go,
or use `ab` or [`siege`][siege]


[boom]: https://github.com/rakyll/boom
[vegeta]: https://github.com/tsenart/vegeta
[siege]: http://www.joedog.org/siege-home/
