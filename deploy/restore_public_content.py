#!/usr/bin/env python
"""Restore only archived hall-of-fame entries, NOTICE posts and their referenced assets.
Python 2.7/3 compatible. Plan first; apply requires the exact saved plan digest.
Never imports members, orders, credentials, comments or likes; never overwrites rows/files.
"""
from __future__ import print_function
import argparse, binascii, hashlib, json, os, posixpath, pwd, re, shutil, subprocess, tarfile, time
try:
    from urllib.parse import urlsplit, unquote, quote
    from html import unescape
except ImportError:
    from urlparse import urlsplit
    from urllib import unquote, quote
    from HTMLParser import HTMLParser
    unescape = HTMLParser().unescape

SOURCE='daeil_legacy_launch'
TARGET='daeil_app_launch'
ARCHIVE='/app/legacy-archives/20261111T144150Z'
DEFAULTS='/root/launch-dba-corrected.cnf'
TABLES=['hall_of_fame_import_batch','hall_of_fame_entry','WEO_BOARDBBS','WEO_FILES']
POST_PRIVATE_FIELDS=['REG_ID','REG_EMAIL','REG_TEL','REG_PWD','REG_IPADDR','RE_ID']


def ident(value):
    if not re.match(r'^[A-Za-z_][A-Za-z0-9_]*$',value): raise ValueError('unsafe identifier')
    return '`'+value+'`'

def literal(value):
    if value is None: return 'NULL'
    raw=str(value).encode('utf8') if not isinstance(value, type(u'')) else value.encode('utf8')
    return "CONVERT(0x"+binascii.hexlify(raw).decode('ascii')+" USING utf8mb4)" if raw else "''"

def query(sql, database=None):
    args=['mysql','--defaults-file='+DEFAULTS,'--default-character-set=utf8mb4','--batch','--skip-column-names']
    if database: args.append(database)
    p=subprocess.Popen(args,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    out,err=p.communicate(sql.encode('utf8'))
    if p.returncode: raise RuntimeError('mysql operation failed: '+err.decode('utf8','replace'))
    return out.decode('utf8').strip()

def columns(table):
    return query("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA="+literal(TARGET)+" AND TABLE_NAME="+literal(table)+" ORDER BY ORDINAL_POSITION").splitlines()

def rows(table,where='1',database=SOURCE):
    cols=columns(table)
    expr=["IF("+ident(c)+" IS NULL,'N',CONCAT('V',HEX(CAST("+ident(c)+" AS CHAR))))" for c in cols]
    raw=query('SELECT '+','.join(expr)+' FROM '+ident(database)+'.'+ident(table)+' WHERE '+where)
    result=[]
    for line in raw.splitlines():
        values=line.split('\t')
        if len(values)!=len(cols): raise ValueError('invalid row encoding')
        result.append(dict(zip(cols,[None if v=='N' else binascii.unhexlify(v[1:]).decode('utf8') for v in values])))
    return result

def digest(path):
    h=hashlib.sha256()
    with open(path,'rb') as f:
        for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
    return h.hexdigest()

def save(path,obj):
    fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,'w') as f:json.dump(obj,f,ensure_ascii=True,indent=2,sort_keys=True)

def asset(value):
    value=unescape(value).strip()
    parsed=urlsplit(value)
    if parsed.netloc and parsed.hostname not in ['daeilfoundation.or.kr','www.daeilfoundation.or.kr','211.169.249.240']:return None
    path=unquote(parsed.path)
    if not isinstance(path,type(u'')):path=path.decode('utf8')
    path=posixpath.normpath('/'+path.lstrip('/'))
    for prefix,kind in [('/old/upload/','legacy'),('/upload/','legacy'),('/files/','legacy'),('/uploads/','uploads')]:
        if path.startswith(prefix):
            relative=path[len(prefix):]
            if not relative or '\\' in relative or '\0' in relative or any(p in ['','.','..'] for p in relative.split('/')):raise ValueError('unsafe asset')
            target=('/files/' if kind=='legacy' else '/uploads/')+relative
            return kind,relative,target
    return None

URL_PATTERN=re.compile(r'''(?:https?://[^\s<>"'()]+|(?:(?:/old)?/upload(?:s)?/|/files/)[^\s<>"'()]+)''',re.I)

def prepare(directory):
    if os.path.exists(directory):raise ValueError('refuse reuse of maintenance directory')
    os.makedirs(directory,0o700)
    manifest=json.load(open(ARCHIVE+'/manifest.json'))
    if not manifest.get('complete'):raise ValueError('incomplete archive')
    current={t:int(query('SELECT COUNT(*) FROM '+ident(TARGET)+'.'+ident(t))) for t in TABLES}
    if current['hall_of_fame_entry'] or current['WEO_BOARDBBS'] or current['hall_of_fame_import_batch']:raise ValueError('target content is not empty; manual conflict review required')
    owner=query("SELECT USR_ID FROM "+TARGET+".WEO_MEMBER WHERE USR_SEQ=1")
    if owner!='jerryhwang018':raise ValueError('administrator mapping changed')
    source_owner=query("SELECT USR_ID FROM "+SOURCE+".WEO_MEMBER WHERE USR_SEQ=12813")
    if source_owner!=owner:raise ValueError('source administrator mismatch')
    data={t:rows(t,"GATE='NOTICE'" if t=='WEO_BOARDBBS' else '1') for t in TABLES if t!='WEO_FILES'}
    posts=data['WEO_BOARDBBS'];seqs=','.join(str(int(p['SEQ'])) for p in posts)
    files=rows('WEO_FILES',"F_GATE IN ('B','BB') AND F_JOIN_SEQ IN ("+seqs+")")
    old_files=[dict(f) for f in files]
    # A disjoint id range prevents archive file #3 replacing current profile #3.
    existing=set(int(v) for v in query('SELECT F_SEQ FROM '+TARGET+'.WEO_FILES').splitlines())
    needed={};missing=[]
    archive_by_kind={}
    for spec in manifest['files']:
        kind='uploads' if spec['source_root']=='/var/www/uploads' else 'legacy'
        archive_by_kind[kind]=spec
    index={kind:{e['path']:e for e in spec['entries']} for kind,spec in archive_by_kind.items()}
    def register(value):
        found=asset(value)
        if not found:return value
        kind,rel,target=found
        entry=index.get(kind,{}).get(rel)
        if entry is None:
            missing.append({'url':value,'kind':kind,'path':rel});return value
        needed[kind+'/'+rel]={'kind':kind,'path':rel,'url':target,'sha256':entry['sha256'],'bytes':entry['bytes'],'archive':archive_by_kind[kind]['archive']}
        return target
    def rewrite(value):
        if not value:return value
        return URL_PATTERN.sub(lambda match:register(match.group(0)),value)
    categories={r.split('\t')[0]:r.split('\t')[1] for r in query('SELECT FC_SEQ,FC_CODE FROM '+SOURCE+'.ALUMNI_FEED_CATEGORY').splitlines()}
    targets={r.split('\t')[1]:r.split('\t')[0] for r in query('SELECT FC_SEQ,FC_CODE FROM '+TARGET+'.ALUMNI_FEED_CATEGORY').splitlines()}
    for post in posts:
        for field in ['CONTENTS','CONTENTS_MD','THUMBNAIL_URL','RE_CONTENTS','RE_FILES']:post[field]=rewrite(post.get(field))
        post['USR_SEQ']='1' if post['USR_SEQ']=='12813' else None
        for field in POST_PRIVATE_FIELDS:post[field]=None
        if post.get('FEED_CATEGORY_SEQ') is not None:
            code=categories.get(post['FEED_CATEGORY_SEQ'])
            if code not in targets:raise ValueError('missing feed category')
            post['FEED_CATEGORY_SEQ']=targets[code]
        if post.get('FILES'):raise ValueError('non-empty legacy FILES requires explicit mapping')
    for f in files:
        original=int(f['F_SEQ']);new=10000+original
        if new in existing:raise ValueError('new file id collides')
        f['F_SEQ']=str(new)  # Preserve inline-image (B) versus attachment (BB) semantics.
        value=f['FILE_PATH'] or ''
        if posixpath.basename(urlsplit(value).path)!=f['FILE_NAME']:value=value.rstrip('/')+'/'+(f['FILE_NAME'] or '')
        f['FILE_PATH']=register(value)
    data['WEO_FILES']=files
    for batch in data['hall_of_fame_import_batch']:
        if batch['uploaded_by'] not in ['0','12813']:raise ValueError('unknown historical batch uploader')
        batch['uploaded_by']='1' if batch['uploaded_by']=='12813' else '0'
    plan={'source':SOURCE,'target':TARGET,'archive':ARCHIVE,'created_at':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),'before_counts':current,'counts':{t:len(v) for t,v in data.items()},'public_posts':sum(p['OPEN_YN']=='Y' for p in posts),'public_hall':sum(p['visible']=='1' and p['consent_public']=='1' and p['tier_man_won'] is not None and p['deleted_at'] is None for p in data['hall_of_fame_entry']),'assets':sorted(needed.values(),key=lambda e:(e['kind'],e['path'])),'missing':missing,'data':data,'original_files':old_files}
    save(directory+'/plan.json',plan)
    print(json.dumps({'plan_sha256':digest(directory+'/plan.json'),'counts':plan['counts'],'public_posts':plan['public_posts'],'public_hall':plan['public_hall'],'asset_count':len(needed),'asset_bytes':sum(e['bytes'] for e in needed.values()),'missing':missing}))

def load_plan(directory, expected):
    if digest(directory+'/plan.json')!=expected:raise ValueError('plan digest mismatch')
    plan=json.load(open(directory+'/plan.json'))
    if plan['missing'] or plan['target']!=TARGET:raise ValueError('invalid plan')
    return plan

def insert_sql(plan,database):
    statements=[]
    for table in TABLES:
        cols=columns(table)
        for row in plan['data'][table]:
            statements.append('INSERT INTO '+ident(database)+'.'+ident(table)+' ('+','.join(ident(c) for c in cols)+') VALUES ('+','.join(literal(row[c]) for c in cols)+');')
    return '\n'.join(statements)

def verify_rows(plan,database,allow_views=False):
    keys={'hall_of_fame_import_batch':'id','hall_of_fame_entry':'id','WEO_BOARDBBS':'SEQ','WEO_FILES':'F_SEQ'}
    for table in TABLES:
        key=keys[table];expected=plan['data'][table]
        where=ident(key)+' IN ('+','.join(literal(r[key]) for r in expected)+')'
        actual=rows(table,where,database)
        if allow_views and table=='WEO_BOARDBBS':
            by_id={r[key]:r for r in expected}
            for row in actual:
                old=by_id[row[key]]
                if int(row['HIT'] or 0)<int(old['HIT'] or 0):raise ValueError('view count decreased')
                row['HIT']=old['HIT']
        normalized=lambda items:sorted(items,key=lambda r:r[key])
        if normalized(actual)!=normalized(expected):raise ValueError('restored row mismatch: '+table)

def stage(directory,plan):
    wanted={a['kind']+'/'+a['path']:a for a in plan['assets']};found=set()
    for archive in sorted(set(a['archive'] for a in wanted.values())):
        with tarfile.open(ARCHIVE+'/'+archive,'r|gz') as tar:
            for member in tar:
                matches=[(k,a) for k,a in wanted.items() if a['archive']==archive and member.name=='files/'+a['path']]
                for key,a in matches:
                    if not member.isfile():raise ValueError('selected asset is not a regular file')
                    target=directory+'/staged/'+key
                    if not os.path.isdir(os.path.dirname(target)):os.makedirs(os.path.dirname(target),0o700)
                    with open(target,'wb') as out:shutil.copyfileobj(tar.extractfile(member),out)
                    os.chmod(target,0o600)
                    if os.path.getsize(target)!=a['bytes'] or digest(target)!=a['sha256']:raise ValueError('staged asset checksum mismatch')
                    found.add(key)
    if found!=set(wanted):raise ValueError('missing staged asset')
    print(json.dumps({'staged_assets':len(found)}))

def rehearse(directory,plan):
    database='daeil_content_restore_check_20261113'
    if query('SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='+literal(database)):raise ValueError('rehearsal database already exists')
    query('CREATE DATABASE '+ident(database)+' CHARACTER SET utf8mb4')
    for table in TABLES:query('CREATE TABLE '+ident(database)+'.'+ident(table)+' LIKE '+ident(TARGET)+'.'+ident(table))
    query('START TRANSACTION;\n'+insert_sql(plan,database)+'\nCOMMIT;')
    verify_rows(plan,database)
    save(directory+'/rehearsal.json',{'verified':True,'database':database,'plan_sha256':digest(directory+'/plan.json')})
    print(json.dumps({'rehearsal_verified':True,'counts':plan['counts']}))

def apply(directory,plan):
    if os.path.exists(directory+'/applied.json'):raise ValueError('already applied')
    rehearsal=json.load(open(directory+'/rehearsal.json'))
    if not rehearsal['verified'] or rehearsal['plan_sha256']!=digest(directory+'/plan.json'):raise ValueError('rehearsal required')
    for t in TABLES:
        if int(query('SELECT COUNT(*) FROM '+ident(TARGET)+'.'+ident(t)))!=plan['before_counts'][t]:raise ValueError('target changed since planning')
    # Capture current profile metadata exactly; these rows must survive untouched.
    original=rows('WEO_FILES','1',TARGET)
    backup=directory+'/before.sql.gz'
    if os.path.exists(backup):raise ValueError('backup already exists')
    import gzip
    dump=subprocess.Popen(['mysqldump','--defaults-file='+DEFAULTS,'--default-character-set=utf8mb4','--lock-tables',TARGET],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    with gzip.open(backup,'wb') as out:shutil.copyfileobj(dump.stdout,out)
    err=dump.stderr.read();code=dump.wait();os.chmod(backup,0o600)
    if code:raise RuntimeError('backup failed')
    if not os.path.getsize(backup):raise ValueError('empty backup')
    account=pwd.getpwnam('alumni-backend');created=[]
    for a in plan['assets']:
        source=directory+'/staged/'+a['kind']+'/'+a['path']
        if digest(source)!=a['sha256']:raise ValueError('staging changed')
        root='/var/www/daeil-launch/'+a['kind'];target=root+'/'+a['path']
        if not os.path.realpath(target).startswith(root+'/'):raise ValueError('unsafe destination')
        if os.path.exists(target):
            if digest(target)!=a['sha256']:raise ValueError('refuse asset overwrite')
            continue
        parent=os.path.dirname(target)
        if not os.path.isdir(parent):os.makedirs(parent,0o755)
        # Exclusive creation protects existing files, including concurrent uploads.
        fd=os.open(target,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o644)
        with os.fdopen(fd,'wb') as out:
            with open(source,'rb') as inp:shutil.copyfileobj(inp,out)
        os.chown(target,account.pw_uid,account.pw_gid);created.append(target)
        if digest(target)!=a['sha256']:raise ValueError('published asset checksum mismatch')
    save(directory+'/created-assets.json',created)
    query('START TRANSACTION;\n'+insert_sql(plan,TARGET)+'\nCOMMIT;')
    verify_rows(plan,TARGET)
    for old in original:
        actual=rows('WEO_FILES','F_SEQ='+literal(old['F_SEQ']),TARGET)
        if actual!=[old]:raise ValueError('existing profile metadata changed')
    save(directory+'/applied.json',{'verified':True,'counts':plan['counts'],'assets':len(plan['assets']),'backup_sha256':digest(backup),'plan_sha256':digest(directory+'/plan.json')})
    print(json.dumps({'restored':plan['counts'],'assets':len(plan['assets']),'existing_profiles_preserved':len(original),'backup_verified':True}))

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('mode',choices=['plan','stage','rehearse','apply','verify']);p.add_argument('directory');p.add_argument('--sha256');a=p.parse_args()
    if a.mode=='plan':prepare(a.directory)
    else:
        plan=load_plan(a.directory,a.sha256)
        if a.mode=='stage':stage(a.directory,plan)
        elif a.mode=='rehearse':rehearse(a.directory,plan)
        elif a.mode=='apply':apply(a.directory,plan)
        else:verify_rows(plan,TARGET,allow_views=True);print('{"rows_verified":true}')
