package main

/*
from External_command import External_command

class Radius_client(External_command):
    global_config = nil
    _avpair_names = ('call-id', 'h323-session-protocol', 'h323-ivr-out', 'h323-incoming-conf-id', \
      'release-source', 'alert-timepoint', 'provisional-timepoint')
    _cisco_vsa_names = ('h323-remote-address', 'h323-conf-id', 'h323-setup-time', 'h323-call-origin', \
      'h323-call-type', 'h323-connect-time', 'h323-disconnect-time', 'h323-disconnect-cause', \
      'h323-voice-quality', 'h323-credit-time', 'h323-return-code', 'h323-redirect-number', \
      'h323-preferred-lang', 'h323-billing-model', 'h323-currency')

    def __init__(s, global_config = {}):
        s.global_config = global_config
        command = global_config.getdefault('radiusclient', '/usr/local/sbin/radiusclient')
        config = global_config.getdefault('radiusclient.conf', nil)
        max_workers = global_config.getdefault('max_radiusclients', 20)
        if config != nil:
            External_command.__init__(s, (command, '-f', config, '-s'), max_workers = max_workers)
        else:
            External_command.__init__(s, (command, '-s'), max_workers = max_workers)

    def _prepare_attributes(s, type, attributes):
        data = [type]
        for a, v in attributes:
            if a in s._avpair_names:
                v = '%s=%s' % (str(a), str(v))
                a = 'Cisco-AVPair'
            elif a in s._cisco_vsa_names:
                v = '%s=%s' % (str(a), str(v))
            data.append('%s="%s"' % (str(a), str(v)))
        return data

    def do_auth(s, attributes, result_callback, *callback_parameters):
        return External_command.process_command(s, s._prepare_attributes('AUTH', attributes), result_callback, *callback_parameters)

    def do_acct(s, attributes, result_callback = nil, *callback_parameters):
        External_command.process_command(s, s._prepare_attributes('ACCT', attributes), result_callback, *callback_parameters)

    def process_result(s, result_callback, result, *callback_parameters):
        if result_callback == nil:
            return
        nav = []
        for av in result[:-1]:
            a, v = [x.strip() for x in av.split(' = ', 1)]
            v = v.strip('\'')
            if (a == 'Cisco-AVPair' or a in s._cisco_vsa_names):
                t = v.split('=', 1)
                if len(t) > 1:
                    a, v = t
            elif v.startswith(a + '='):
                v = v[len(a) + 1:]
            nav.append((a, v))
        External_command.process_result(s, result_callback, (tuple(nav), int(result[-1])), *callback_parameters)
*/
