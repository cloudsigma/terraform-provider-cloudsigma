"""Fail-closed semantic evidence; raw output remains provider-redactable display.

Official Go test2json executes trusted test binaries in reserved framing mode.
Output payload is never semantic evidence. Framing is not a hostile-code boundary.
"""
import hashlib
import json
import pathlib
import re

PLAN = json.loads(pathlib.Path(__file__).with_name('offline_plan.json').read_text())


def semantic(record, output, index):
    if index >= len(PLAN):
        raise ValueError('extra stage')
    expected = PLAN[index]
    if set(record) != {'package', 'stage', 'argv', 'exit'}:
        raise ValueError('malformed stage')
    if (record['package'], record['stage']) != (expected['package'], expected['stage']):
        raise ValueError('stage order mismatch')
    if type(record['exit']) is not int or record['exit'] != 0:
        raise ValueError('stage exit nonzero or malformed')
    argv = record['argv']
    if not isinstance(argv, list) or not all(type(a) is str for a in argv):
        raise ValueError('malformed argv')
    if expected['stage'] == 'build':
        if len(argv) != 9 or argv[:4] != ['go', 'test', '-vet=off', '-overlay'] or argv[5] != '-c' or argv[6] != '-o' or argv[8] != './'+expected['package']:
            raise ValueError('build selection mismatch')
        return dict(expected, exit=0, passed=0)
    names = expected['tests']
    selector = '^('+'|'.join(names)+')$' if expected['stage'] == 'suite' else '^'+names[0]+'$'
    tail = ['-test.count=1'] if expected['stage'] == 'suite' else ([] if expected['stage'] == 'canary' else ['-test.timeout','590s'])
    if argv[:5] != ['go','tool','test2json','-p',expected['package']] or argv[6:] != ['-test.run', selector, '-test.v=test2json']+tail:
        raise ValueError('test selection mismatch')
    return validate_events(expected, output)


def validate_events(expected, output):
    names = set(expected['tests'])
    running, ended, tops = set(), {}, set()
    started = finished = False
    for line in output.splitlines():
        try:
            event = json.loads(line)
        except (ValueError, UnicodeError):
            raise ValueError('invalid event') from None
        if not isinstance(event, dict) or event.get('Package') != expected['package']:
            raise ValueError('event package mismatch')
        action, name = event.get('Action'), event.get('Test')
        if finished:
            raise ValueError('event after package terminal')
        if action == 'start' and not name:
            if started: raise ValueError('duplicate start')
            started = True
            continue
        if not started: raise ValueError('missing package start')
        if action == 'output': continue  # Never inspect or publish arbitrary Output as semantics.
        if not name:
            if action != 'pass' or running or set(ended) == set() or tops != names:
                raise ValueError('invalid package terminal')
            finished = True
            continue
        if not isinstance(name, str) or name.split('/')[0] not in names:
            raise ValueError('unselected event')
        if action == 'run':
            if name in running or name in ended: raise ValueError('duplicate run')
            if '/' in name and not any(name.startswith(parent+'/') for parent in running): raise ValueError('missing parent')
            running.add(name)
            if '/' not in name: tops.add(name)
        elif action in ('pass','skip','fail'):
            if name not in running or any(n.startswith(name+'/') for n in running): raise ValueError('terminal order')
            running.remove(name)
            ended[name] = action
        elif action in ('pause','cont'):
            if name not in running: raise ValueError('inactive test')
        else:
            raise ValueError('unexpected action')
    if not finished or running or len(ended) != expected['total'] or 'fail' in ended.values():
        raise ValueError('incomplete or failed events')
    # Only static allowlisted top-level names and aggregate status counts leave here.
    return dict(expected, exit=0, passed=list(ended.values()).count('pass'), skipped=list(ended.values()).count('skip'))


def publish_stage(directory, record):
    directory = pathlib.Path(directory)
    receipt = directory/'execution.json'
    if directory.is_symlink() or receipt.is_symlink():
        raise ValueError('unsafe receipt')
    records = json.loads(receipt.read_text())
    if not isinstance(records, list) or not records or records[-1] != record:
        raise ValueError('receipt mismatch')
    index = len(records)-1
    if index >= len(PLAN) or (record.get('package'), record.get('stage')) != (PLAN[index]['package'], PLAN[index]['stage']):
        raise ValueError('stage mismatch')
    name = PLAN[index]['package'].replace('/','-')+'-'+PLAN[index]['stage']+'.log'
    log = directory/name
    if log.is_symlink() or not log.is_file():
        raise ValueError('unsafe log')
    output = log.read_bytes()
    # Do not JSON-escape, encode or otherwise transform payload for publication.
    # Prefix physical lines to prevent workflow command interpretation; provider
    # masking remains authoritative and display is explicitly not lossless.
    print('OFFLINE_PROVENANCE '+json.dumps(dict(stage=index, scope='RAW_PRE_PROVIDER', sha256=hashlib.sha256(output).hexdigest(), bytes=len(output))), flush=True)
    print('OFFLINE_DISPLAY PROVIDER_REDACTED_NOT_LOSSLESS', flush=True)
    if record['stage'] == 'build':
        display = output.decode('utf-8', errors='replace')
    else:
        try:
            display = ''.join(e.get('Output','') for e in map(json.loads, output.splitlines()) if e.get('Action') == 'output')
        except (ValueError, TypeError, AttributeError):
            display = 'MALFORMED_EVENT_STREAM'
    for line in display.splitlines():
        print('OFFLINE_OUTPUT '+line, flush=True)
    try:
        result = semantic(record, output, index)
    except ValueError:
        print('OFFLINE_SEMANTIC '+json.dumps(dict(stage=index, valid=False, reason='EVIDENCE_REFUSED')), flush=True)
        raise
    print('OFFLINE_SEMANTIC '+json.dumps(dict(result, valid=True)), flush=True)


def finish(directory, records):
    if len(records) != len(PLAN):
        raise ValueError('missing stages')
    for i, record in enumerate(records):
        name = PLAN[i]['package'].replace('/','-')+'-'+PLAN[i]['stage']+'.log'
        semantic(record, (pathlib.Path(directory)/name).read_bytes(), i)
    print('OFFLINE_COMPLETE '+json.dumps(dict(stages=18, valid=True)), flush=True)
