var assert = require('assert');

var Confdis = require('../index');

var c;
var rootKey = 'simpletest';
var dummyData = require('./dummyData.json');

describe('Confdis', function() {

    describe('test constructor & opts', function() {

        it('should initiate with error', function() {
            var y = new Confdis();
            assert(y instanceof Error);
        });

        it('should initiate without error', function() {

            c = new Confdis({
                rootKey: rootKey,
                host: process.env.TEST_HOST,
                port: process.env.TEST_PORT,
                subscribe_to_changes: true
            });

            assert(c instanceof Confdis);
            assert(c instanceof Object);

            describe('test connection', function() {

                it('it should connect to redis & fire events', function(done) {
                    c.on('ready', function() {
                        done();
                    });

                    c.on('error', function(err) {
                        done(err);
                    });

                    c.connect(function(err) {
                        assert(!err);
                    });

                });

                it('it should save with error - config empty', function(done) {
                    c.save(function(err) {
                        assert(err);
                        done();
                    });
                });

                it('it should sync with error - config empty', function(done) {
                    c.sync(function(err, config, changes) {
                        assert(err);
                        done();
                    });
                });


                it('it should save the dummy data', function(done) {
                    c.config = dummyData;
                    c.save(function(err) {
                        assert(err === undefined);
                        assert(c.config);
                        done(err);

                    });
                });

                it('should load & sync the dummy data', function(done) {
                    c.sync(function(err, config, changes) {
                        assert(err === null);
                        assert(config);
                        done(err);
                    });
                });

                it('it should modify the config', function(done) {
                    c.config.version = 99;
                    c.save(function(err) {
                        assert(err === undefined);
                        assert(c.config);
                        done(err);
                    });
                });

                it('verify integrity of the dummy data', function() {
                    assert(c.config.version === 99);
                    assert(c.config.properties["molecular mass"] === 30.0690);
                    assert(c.config.atoms.coords["3d"].indexOf(1.166929) >= 0);

                    // reset for next changes on sync test
                    c.config.version = 0;
                });

                it('it should give me a list of changes on sync', function(done) {
                    c.sync(function(err, config, changes) {
                        assert(err === null);
                        assert(changes.version);
                        assert(JSON.stringify(changes.version) === JSON.stringify([0, 99]));
                        done(err);
                    });
                });

                it('it should give me a local value', function(done) {
                    c.getValue('atoms', function(err, value){
                        assert(!err);
                        assert(value.ids);
                        done();
                    });
                });

                it('it should set a local value and emit a pubsub change', function(done) {

                    c.on('pubsub-message', function(channel, message){
                        var change = JSON.parse(message);
                        assert(change['test-key'] === 'test-value');
                        done();
                    });

                    c.setValue('test-key', 'test-value', function(err, value){
                        assert(!err);
                        c.getValue('test-key', function(err, value) {
                          assert(!err);
                          assert(value === 'test-value');
                        });
                    });

                });

                it('it should emit an event on sync', function(done) {

                    c.on('sync', function(err, config, changes) {
                        done();
                    });
                    c.on('sync-error', function(err) {
                        done(err);
                    });

                    c.sync(function(err, config, changes) {});

                });

                it('is should give me a component value', function(done) {
                    c.getComponentValue(rootKey, "atoms", function(err, val){
                        assert(!err);
                        assert(val);
                        assert(val.ids);
                        done();
                    });
                });

                it('is should give me a error for a nonexistent component value', function(done) {
                    c.getComponentValue(Math.random().toString(36).substring(7), "atoms", function(err, val){
                        assert(err);
                        assert(!val);
                        done();
                    });
                });

                it('should clear the config data in redis & memory', function(done) {
                    c.clear(function(err) {
                        assert(!c.config);
                        assert(!err);
                        done();
                    });
                });

            });
        });
    });
});

