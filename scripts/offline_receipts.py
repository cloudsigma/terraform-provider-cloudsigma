"""Publish only the current offline stage receipt and its complete captured output.

No directory glob, environment dump, or artifact upload. Prefix every output line
so test output cannot become a GitHub workflow command. Synthetic PRIVATE
canaries are evidence, not credentials: retain them and all failure diagnostics.
GitHub's existing secret masking still applies to hosted stdout.
"""
import hashlib
import json
import pathlib
import re


def publish_stage(directory, record):
    directory = pathlib.Path(directory)
    if directory.is_symlink():
        raise ValueError('symlink receipt directory')
    receipt = directory / 'execution.json'
    if receipt.is_symlink():
        raise ValueError('symlink execution receipt')
    records = json.loads(receipt.read_text())
    if not isinstance(records, list) or not records or records[-1] != record:
        raise ValueError('missing or mismatched current receipt')
    if set(record) != {'package', 'stage', 'argv', 'exit'}:
        raise ValueError('malformed receipt fields')
    if record['package'] not in ('internal/cloudsigmaapi', 'cloudsigma', 'internal/provider', 'readonly'):
        raise ValueError('unknown package')
    if not isinstance(record['stage'], str) or not re.fullmatch(r'(build|canary|suite|TestNative(?:26(?:Dispatch|AckCancel|RichSuccess|AttachmentFaults)|28(?:Matrix|SnapshotLifecycle)))', record['stage']):
        raise ValueError('unknown stage')
    if type(record['exit']) is not int or not isinstance(record['argv'], list) or not record['argv'] or not all(isinstance(arg, str) for arg in record['argv']):
        raise ValueError('malformed command or exit')
    name = record['package'].replace('/', '-') + '-' + record['stage'] + '.log'
    log = directory / name
    if log.is_symlink() or not log.is_file():
        raise ValueError('missing or unsafe stage log')
    output = log.read_bytes()
    manifest = dict(record, log=name, bytes=len(output), sha256=hashlib.sha256(output).hexdigest())
    print('OFFLINE_STAGE ' + json.dumps(manifest, sort_keys=True), flush=True)
    # JSON strings preserve every byte (Latin-1 mapping), including CR, escape
    # sequences and missing final newline; each bounded chunk is reversible.
    # No truncation: the measured full 18-stage output is only ~430 KB.
    for offset in range(0, len(output), 4096):
        print('OFFLINE_OUTPUT ' + json.dumps(output[offset:offset + 4096].decode('latin1')), flush=True)
    print('OFFLINE_END ' + name, flush=True)
