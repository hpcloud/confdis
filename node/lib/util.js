/**
 * Copyright (c) ActiveState 2013 - ALL RIGHTS RESERVED.
 */

'use strict';

var Os = require('os');

/**
 * Returns the IPv4/v6 address for the requested interface
 * @param {string} iface - i.e. eth0, eth1
 * @callback {Error, Object} cb
 */
module.exports.getInterfaceAddress = function (iface) {
    if (!iface) { throw new Error('Interface required must be the first argument'); }

    var ifaces = Os.networkInterfaces();
    var ipv4Addr = null;
    var ipv6Addr = null;

    if (iface in ifaces) {
        for (var address in ifaces[iface]) {
            if (ifaces[iface][address].family === 'IPv4') {
                ipv4Addr = ifaces[iface][address].address;
            } else if (ifaces[iface][address].family === 'IPv6') {
                ipv6Addr = ifaces[iface][address].address;
            }
        }

        var ipAddr = ipv4Addr || ipv6Addr;
        return null, { address: ipAddr, IPv4Address: ipv4Addr, IPv6Address: ipv6Addr };

    }else{
        throw new Error('Interface ' + iface + ' is not available');
    }
};

