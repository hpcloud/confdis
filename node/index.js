(function() {
    'use strict';

    var events = require('events'),
        redis = require('redis');

    var Confdis = function(opts) {

        if (!opts) return new Error('Options object not supplied');

        this.opts = opts;

        var requiredOpts = ['host', 'port', 'rootKey'];

        requiredOpts.forEach(function(val, i, arr) {
            if (!opts[val]) {
                return new Error(val + ' option required but not specified');
            }
        });

        this.redisHost = opts.host;
        this.redisPort = opts.port;

        this.config = null;
        this.db = null;
        this.pubsubDB = null;
        this.rootKey = null;

        this._PUB_SUFFIX = ":_changes";
        this._REDIS_CONNECT_MAX_ATTEMPTS = null;
        this._REDIS_CONNECT_TIMEOUT = false;
        this._REDIS_OFFLINE_QUEUE = false;

    };

    Confdis.prototype = new events.EventEmitter();

    Confdis.prototype.redisConnectOpts = function() {
        return {
            connect_timeout: this._REDIS_CONNECT_TIMEOUT,
            enable_offline_queue: this._REDIS_OFFLINE_QUEUE,
            max_attempts: this._REDIS_CONNECT_MAX_ATTEMPTS
        };
    };

    Confdis.prototype.connect = function(cb) {

        var self = this;

        if (!cb) return new Error('Callback function not supplied as last argument');

        if (!self.redisHost || !self.redisPort) {
            return cb(new Error('Host and port values must be provided'));
        }


        this.db = redis.createClient(self.redisPort, self.redisHost, this.redisConnectOpts());

        this.db.on('error', function(err) {
            self.emit('error', err);
            return cb(err);
        });

        this.db.on('ready', function() {
            self.emit('ready');
            return cb();
        });

    };

    Confdis.prototype.sync = function(cb) {
        var self = this;

        self.db.get(self.rootKey, function(err, reply) {
            if (!err) {
                return cb(null, reply);
            } else {
                self.emit('sync-error', err);
                return cb(err);
            }
        });
    };

    Confdis.prototype.save = function(cb) {

        var self = this;

        if (self.config) {
            self.db.set(self.rootKey, self.config, function(err, res) {
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

    Confdis.prototype.subscribe = function(cb) {
        var self = this;

        // Need multiple connections for subscriber mode
        self.pubsubDB = redis.createClient(self.redisHost, self.redisPort, self.redisConnectOpts());

        self.pubsubDB.on('error', function(err) {
            self.emit('error', err);
            return cb(err);
        });

        self.pubsubDB.on('ready', function() {
            self.pubsubDB.subscribe(self.rootKey + self._PUB_SUFFIX);
        });

        self.pubsubDB.on('subscribe', function(channel, count) {
            self.emit('subscribing', channel);
            return cb();
        });

        self.pubsubDB.on('message', function(channel, message) {
            self.emit('pubsub-message', channel, message);
        });

        self.on('pubsub-message', function(channel, message) {
            self.sync();
        });

    };

    module.exports = Confdis;
})();
