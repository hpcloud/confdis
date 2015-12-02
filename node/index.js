/**
 * Copyright (c) ActiveState 2013 - ALL RIGHTS RESERVED.
 */

(function () {
    'use strict';

    var JsonDiff = require('jsondiffpatch'),
        Events = require('events'),
        Redis = require('redis'),
        Util = require('./lib/util'),
        _ = require('lodash');

    var Confdis = function (opts) {

        if (!(this instanceof Confdis)) {
            return new Confdis(opts);
        }

        if (!opts) { return new Error('Options object not supplied'); }

        this.opts = opts;

        var requiredOpts = ['host', 'port', 'rootKey'];

        requiredOpts.forEach(function (val, i, arr) {
            if (!opts[val]) {
                return new Error(val + ' option required but not specified');
            }
        });

        this.redisHost = opts.host;
        this.redisPort = opts.port;
        this.redisIndex = opts.index || null;
        this.rootKey = opts.rootKey;

        this.config = null;
        this.db = null;
        this.pubsubDB = null;

        this._PUB_SUFFIX = ':_changes';
        this._REDIS_CONNECT_MAX_ATTEMPTS = null;
        this._REDIS_CONNECT_TIMEOUT = 60 * 1000;
        this._REDIS_OFFLINE_QUEUE = true;

        if (this.opts.subscribe_to_changes === true) {
            this.subscribe(function(err) {
                if (err) { throw err; }
            });
        }

        return this;
    };

    Confdis.prototype = Object.create(Events.EventEmitter.prototype);

    Confdis.prototype.redisConnectOpts = function () {
        return {
            connect_timeout: this._REDIS_CONNECT_TIMEOUT,
            enable_offline_queue: this._REDIS_OFFLINE_QUEUE,
            max_attempts: this._REDIS_CONNECT_MAX_ATTEMPTS
        };
    };

    Confdis.prototype.connect = function (cb) {

        var self = this;

        if (!cb) { return new Error('Callback function not supplied as last argument'); }

        if (!self.redisHost || !self.redisPort) {
            return cb(new Error('Host and port values must be provided'));
        }

        /* 103391 Components should avoid redis connection on loopback addresses */
        if (self.redisHost.match(/^(127\.[\d.]+|[0:]+1|localhost)$/i)) {
            self.redisHost = Util.getInterfaceAddress('eth0').address;
        }

        this.db = Redis.createClient(self.redisPort, self.redisHost, this.redisConnectOpts());

        this.db.on('error', function (err) {
            self.emit('error', err);
            // OP 302284
      	    //
      	    // Do __not__ pass the error to the done callback 'cb'.
      	    // This callback, provided by s-rest, will not just print
      	    // the message again, but also kill s-rest by throwing it
      	    // as error.
      	    //
      	    // With this statement gone the emitter, i.e. the redis
      	    // client, will go and start retrying making the
      	    // connection, until it either works, or the limits (See
      	    // connect timeout set by this module) are reached. During
      	    // this time s-rest keeps running instead of forcing
      	    // supervisord to restart it for the connection retry.
        });

        this.db.on('ready', function () {
            self.db.removeAllListeners('error');

            self.db.on('error', function (err) {
                self.emit('error', err);
            });

            self.db.on('reconnecting', function () {
                self.emit('reconnecting');
            });

            if (self.redisIndex !== null) {
                self.db.select(self.redisIndex, function (err) {
                    if (err) { return self.emit('error', err); }
                    self.emit('ready');
                    return cb();
                });
            } else {
                self.emit('ready');
                return cb();
            }
        });

    };

    Confdis.prototype.sync = function (cb) {
        var self = this;

        self.db.get(self.opts.rootKey, function (err, reply) {
            if (!err) {
                if (reply) {
                    var changes = null;

                    if (self.config) {
                        var prevConfig = JSON.parse(JSON.stringify(self.config));
                        self.config = JSON.parse(reply);
                        changes = JsonDiff.diff(prevConfig, self.config);
                    } else {
                        self.config = JSON.parse(reply);
                    }

                    self.emit('sync', changes);

                    if (_.isFunction(cb)) {
                        return cb(null, reply, changes);
                    }
                } else {
                    err = new Error('config is empty, not syncing');
                    self.emit('sync-error', err);
                    if (_.isFunction(cb)) {
                        return cb(err);
                    }
                }
            } else {
                self.emit('sync-error', err);
                if (_.isFunction(cb)) {
                    return cb(err);
                }
            }
        });
    };

    Confdis.prototype.save = function (cb) {

        var self = this;

        if (self.config) {
            self.db.set(self.opts.rootKey, JSON.stringify(self.config), function (err, res) {
                if (err) {
                    self.emit('error', err);
                    return cb(err);
                }
                return cb();
            });
        } else {
            return cb(new Error('config is empty, aborting save'));
        }

    };

    Confdis.prototype.clear = function (cb) {
        var self = this;
        this.db.set(this.opts.rootKey, '', function (err, res) {
            if (!err) {
                self.config = null;
                return cb();
            } else {
                return cb(err);
            }
        });
    };

    Confdis.prototype.setValue = function (key, value, cb) {
        var self = this;
        self.config[key] = value;
        self.save(function(err) {
            if (err) { return cb(err); }
            var change = {};
            change[key] = value;
            self.db.publish(self.rootKey + self._PUB_SUFFIX, JSON.stringify(change), cb);
        });
    };

    Confdis.prototype.getValue = function (key, cb) {
        var self = this;
        if (self.config[key]) {
            return cb ? cb(null, self.config[key]) : self.config[key];
        } else {
            return cb ? cb() : null;
        }
    };

    Confdis.prototype.getComponentValue = function (component, key, cb) {
        this.db.get(component, function (err, reply) {
            if (err || !reply) return cb(err || new Error('Empty config'));
            var componentConf = {};
            try {
                componentConf = JSON.parse(reply);
            } catch (err) {
                return cb(err);
            }
            if (componentConf.hasOwnProperty(key)) {
                return cb(null, componentConf[key]);
            } else {
                return cb(new Error('Key: ' + key + ' not found'));
            }
        });
    };

    Confdis.prototype.subscribe = function (cb) {
        var self = this;

        // Need multiple connections for subscriber mode
        self.pubsubDB = Redis.createClient(self.redisPort, self.redisHost, self.redisConnectOpts());

        self.pubsubDB.once('error', function (err) {
            self.emit('error', err);
            cb(err);
        });

        self.pubsubDB.on('ready', function () {
            self.pubsubDB.removeAllListeners('error');
            self.pubsubDB.on('error', function (err) {
                self.emit('error', err);
            });

            self.pubsubDB.subscribe(self.rootKey + self._PUB_SUFFIX);
        });

        self.pubsubDB.once('subscribe', function (channel, count) {
            self.pubsubDB.removeAllListeners('subscribe');
            self.pubsubDB.on('subscribe', function (err) {
                self.emit('subscribing');
            });
            return cb();
        });

        self.pubsubDB.on('message', function (channel, message) {
            self.emit('pubsub-message', channel, message);
        });

        self.pubsubDB.on('reconnecting', function () {
            self.emit('reconnecting');
        });

        self.on('pubsub-message', function (channel, message) {
            self.sync();
        });

    };

    module.exports = Confdis;
})();
