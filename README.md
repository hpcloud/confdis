# confdis

Go, Ruby and Node library for managing JSON-based configuration in
redis.

## faq

### Why keep all implementations in a single repo?

To keep tests, feature changes, bug fixes (or any future changes) in
sync between all implementations. otherwise, it is easier to forget to
extend a change/fix to other implementations spanning multiple repos.

### Common tests?

Not currently, but that would be ideal; if there was common test data
to be shared between all implementations.

## history

http://bugs.activestate.com/show_bug.cgi?id=98612
