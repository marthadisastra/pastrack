"""Install the requested domain proxy, preserving SSL/challenge includes and backups."""
import os,shutil,subprocess,time
path='/www/server/panel/vhost/nginx/toko.azlaundry.my.id.conf'
rewrite='/www/server/panel/vhost/rewrite/toko.azlaundry.my.id.conf'
backup='/root/pastrack-web-backups/'+time.strftime('%Y%m%d-%H%M%S')
os.makedirs(backup,mode=0o700)
shutil.copy2(path,backup+'/site.conf');shutil.copy2(rewrite,backup+'/rewrite.conf')
config='''server {
    listen 80;
    server_name toko.azlaundry.my.id;
    root /www/wwwroot/pastrack;
    include /www/server/panel/vhost/nginx/well-known/toko.azlaundry.my.id.conf;
    location / { return 301 https://$host$request_uri; }
}
server {
    listen 443 ssl http2;
    server_name toko.azlaundry.my.id;
    root /www/wwwroot/pastrack;
    include /www/server/panel/vhost/nginx/extension/toko.azlaundry.my.id/*.conf;
    #CERT-APPLY-CHECK--START
    include /www/server/panel/vhost/nginx/well-known/toko.azlaundry.my.id.conf;
    #CERT-APPLY-CHECK--END
    #SSL-START
    ssl_certificate /www/server/panel/vhost/cert/toko.azlaundry.my.id/fullchain.pem;
    ssl_certificate_key /www/server/panel/vhost/cert/toko.azlaundry.my.id/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    add_header Strict-Transport-Security "max-age=31536000" always;
    error_page 497 https://$host$request_uri;
    #SSL-END
    #REWRITE-START
    include /www/server/panel/vhost/rewrite/toko.azlaundry.my.id.conf;
    #REWRITE-END
    location = /healthz { return 404; }
    location / {
        proxy_pass http://127.0.0.1:18089;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_connect_timeout 5s;
        proxy_read_timeout 25s;
    }
    access_log /www/wwwlogs/toko.azlaundry.my.id.log;
    error_log /www/wwwlogs/toko.azlaundry.my.id.error.log;
}
'''
try:
    with open(path,'w') as f:f.write(config)
    with open(rewrite,'w') as f:f.write('# Routing is handled by the Pastrack Beego application.\n')
    subprocess.run(['/www/server/nginx/sbin/nginx','-t'],check=True)
    subprocess.run(['/www/server/nginx/sbin/nginx','-s','reload'],check=True)
except Exception:
    shutil.copy2(backup+'/site.conf',path);shutil.copy2(backup+'/rewrite.conf',rewrite)
    raise
print('HTTPS_PROXY_CONFIGURED; backup:',backup)
