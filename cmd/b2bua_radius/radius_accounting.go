// Copyright (c) 2003-2005 Maxim Sobolev. All rights reserved.
// Copyright (c) 2006-2014 Sippy Software, Inc. All rights reserved.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package main

/*
from time import time, strftime, gmtime
from Timeout import Timeout

sipErrToH323Err = {400:('7f', 'Interworking, unspecified'), 401:('39', 'Bearer capability not authorized'), \
  402:('15', 'Call rejected'), 403:('39', 'Bearer capability not authorized'), 404:('1', 'Unallocated number'), \
  405:('7f', 'Interworking, unspecified'), 406:('7f', 'Interworking, unspecified'), 407:('15', 'Call rejected'), \
  408:('66', 'Recover on Expires timeout'), 409:('29', 'Temporary failure'), 410:('1', 'Unallocated number'), \
  411:('7f', 'Interworking, unspecified'), 413:('7f', 'Interworking, unspecified'), 414:('7f', 'Interworking, unspecified'), \
  415:('4f', 'Service or option not implemented'), 420:('7f', 'Interworking, unspecified'), 480:('12', 'No user response'), \
  481:('7f', 'Interworking, unspecified'), 482:('7f', 'Interworking, unspecified'), 483:('7f', 'Interworking, unspecified'), \
  484:('1c', 'Address incomplete'), 485:('1', 'Unallocated number'), 486:('11', 'User busy'), 487:('12', 'No user responding'), \
  488:('7f', 'Interworking, unspecified'), 500:('29', 'Temporary failure'), 501:('4f', 'Service or option not implemented'), \
  502:('26', 'Network out of order'), 503:('3f', 'Service or option unavailable'), 504:('66', 'Recover on Expires timeout'), \
  505:('7f', 'Interworking, unspecified'), 580:('2f', 'Resource unavailable, unspecified'), 600:('11', 'User busy'), \
  603:('15', 'Call rejected'), 604:('1',  'Unallocated number'), 606:('3a', 'Bearer capability not presently available')}

class RadiusAccounting(object):
    global_config = nil
    drec = nil
    crec = nil
    iTime = nil
    cTime = nil
    sip_cid = nil
    origin = nil
    lperiod = nil
    el = nil
    send_start = nil
    complete = false
    ms_precision = false
    user_agent = nil
    p1xx_ts = nil
    p100_ts = nil

    def __init__(s, global_config, origin, lperiod = nil, send_start = false):
        s.global_config = global_config
        s._attributes = [('h323-call-origin', origin), ('h323-call-type', 'VoIP'), \
          ('h323-session-protocol', 'sipv2')]
        s.drec = false
        s.crec = false
        s.origin = origin
        s.lperiod = lperiod
        s.send_start = send_start

    def setParams(s, username, caller, callee, h323_cid, sip_cid, remote_ip, \
      h323_in_cid = nil):
        if caller == nil:
            caller = ''
        s._attributes.extend((('User-Name', username), ('Calling-Station-Id', caller), \
          ('Called-Station-Id', callee), ('h323-conf-id', h323_cid), ('call-id', sip_cid), \
          ('Acct-Session-Id', sip_cid), ('h323-remote-address', remote_ip)))
        if h323_in_cid != nil and h323_in_cid != h323_cid:
            s._attributes.append(('h323-incoming-conf-id', h323_in_cid))
        s.sip_cid = str(sip_cid)
        s.complete = true

    def conn(s, ua, rtime, origin):
        if s.crec:
            return
        s.crec = true
        s.iTime = ua.setup_ts
        s.cTime = ua.connect_ts
        if ua.remote_ua != nil and s.user_agent == nil:
            s.user_agent = ua.remote_ua
        if ua.p1xx_ts != nil:
            s.p1xx_ts = ua.p1xx_ts
        if ua.p100_ts != nil:
            s.p100_ts = ua.p100_ts
        if s.send_start:
            s.asend('Start', rtime, origin, ua)
        s._attributes.extend((('h323-voice-quality', 0), ('Acct-Terminate-Cause', 'User-Request')))
        if s.lperiod != nil and s.lperiod > 0:
            s.el = Timeout(s.asend, s.lperiod, -1, 'Alive')

    def disc(s, ua, rtime, origin, result = 0):
        if s.drec:
            return
        s.drec = true
        if s.el != nil:
            s.el.cancel()
            s.el = nil
        if s.iTime == nil:
            s.iTime = ua.setup_ts
        if s.cTime == nil:
            s.cTime = rtime
        if ua.remote_ua != nil and s.user_agent == nil:
            s.user_agent = ua.remote_ua
        if ua.p1xx_ts != nil:
            s.p1xx_ts = ua.p1xx_ts
        if ua.p100_ts != nil:
            s.p100_ts = ua.p100_ts
        s.asend('Stop', rtime, origin, result, ua)

    def asend(s, type, rtime = nil, origin = nil, result = 0, ua = nil):
        if not s.complete:
            return
        if rtime == nil:
            rtime = time()
        if ua != nil:
            duration, delay, connected = ua.getAcct()[:3]
        else:
            # Alive accounting
            duration = rtime - s.cTime
            delay = s.cTime - s.iTime
            connected = true
        if not(s.ms_precision):
            duration = round(duration)
            delay = round(delay)
        attributes = s._attributes[:]
        if type != 'Start':
            if result >= 400:
                try:
                    dc = sipErrToH323Err[result][0]
                except:
                    dc = '7f'
            elif result < 200:
                dc = '10'
            else:
                dc = '0'
            attributes.extend((('h323-disconnect-time', s.ftime(s.iTime + delay + duration)), \
              ('Acct-Session-Time', '%d' % round(duration)), ('h323-disconnect-cause', dc)))
        if type == 'Stop':
            if origin == 'caller':
                release_source = '2'
            elif origin == 'callee':
                release_source = '4'
            else:
                release_source = '8'
            attributes.append(('release-source', release_source))
        attributes.extend((('h323-connect-time', s.ftime(s.iTime + delay)), ('h323-setup-time', s.ftime(s.iTime)), \
          ('Acct-Status-Type', type)))
        if s.user_agent != nil:
            attributes.append(('h323-ivr-out', 'sip_ua:' + s.user_agent))
        if s.p1xx_ts != nil:
            attributes.append(('Acct-Delay-Time', round(s.p1xx_ts)))
        if s.p100_ts != nil:
            attributes.append(('provisional-timepoint', s.ftime(s.p100_ts)))
        pattributes = ['%-32s = \'%s\'\n' % (x[0], str(x[1])) for x in attributes]
        pattributes.insert(0, 'sending Acct %s (%s):\n' % (type, s.origin.capitalize()))
        s.global_config['_sip_logger'].write(call_id = s.sip_cid, *pattributes)
        s.global_config['_radius_client'].do_acct(attributes, s._process_result, s.sip_cid, time())

    def ftime(s, t):
        gt = gmtime(t)
        day = strftime('%d', gt)
        if day[0] == '0':
            day = day[1]
        if s.ms_precision:
            msec = (t % 1) * 1000
        else:
            msec = 0
        return strftime('%%H:%%M:%%S.%.3d GMT %%a %%b %s %%Y' % (msec, day), gt)

    def _process_result(s, results, sip_cid, btime):
        delay = time() - btime
        rcode = results[1]
        if rcode in (0, 1):
            if rcode == 0:
                message = 'Acct/%s request accepted (delay is %.3f)\n' % (s.origin, delay)
            else:
                message = 'Acct/%s request rejected (delay is %.3f)\n' % (s.origin, delay)
        else:
            message = 'Error sending Acct/%s request (delay is %.3f)\n' % (s.origin, delay)
        s.global_config['_sip_logger'].write(message, call_id = sip_cid)
*/
