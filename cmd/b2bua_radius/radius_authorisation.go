package main

/*
from Radius_client import Radius_client
from time import time

class RadiusAuthorisation(Radius_client):
    def do_auth(s, username, caller, callee, h323_cid, sip_cid, remote_ip, res_cb, \
      realm = nil, nonce = nil, uri = nil, response = nil, extra_attributes = nil):
        sip_cid = str(sip_cid)
        attributes = nil
        if nil not in (realm, nonce, uri, response):
            attributes = [('User-Name', username), ('Digest-Realm', realm), \
              ('Digest-Nonce', nonce), ('Digest-Method', 'INVITE'), ('Digest-URI', uri), \
              ('Digest-Algorithm', 'MD5'), ('Digest-User-Name', username), ('Digest-Response', response)]
        else:
            attributes = [('User-Name', remote_ip), ('Password', 'cisco')]
        if caller == nil:
            caller = ''
        attributes.extend((('Calling-Station-Id', caller), ('Called-Station-Id', callee), ('h323-conf-id', h323_cid), \
          ('call-id', sip_cid), ('h323-remote-address', remote_ip), ('h323-session-protocol', 'sipv2')))
        if extra_attributes != nil:
            for a, v in extra_attributes:
                attributes.append((a, v))
        message = 'sending AAA request:\n'
        message += reduce(lambda x, y: x + y, ['%-32s = \'%s\'\n' % (x[0], str(x[1])) for x in attributes])
        s.global_config['_sip_logger'].write(message, call_id = sip_cid)
        Radius_client.do_auth(s, attributes, s._process_result, res_cb, sip_cid, time())

    def _process_result(s, results, res_cb, sip_cid, btime):
        delay = time() - btime
        rcode = results[1]
        if rcode in (0, 1):
            if rcode == 0:
                message = 'AAA request accepted (delay is %.3f), processing response:\n' % delay
            else:
                message = 'AAA request rejected (delay is %.3f), processing response:\n' % delay
            if len(results[0]) > 0:
                message += reduce(lambda x, y: x + y, ['%-32s = \'%s\'\n' % x for x in results[0]])
        else:
            message = 'Error sending AAA request (delay is %.3f)\n' % delay
        s.global_config['_sip_logger'].write(message, call_id = sip_cid)
        res_cb(results)
*/
