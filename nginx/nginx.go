package nginx

import "fmt"

type Config struct {
	ServerName string
	Root string
}

func (c *Config) String() string {
	configFile := fmt.Sprintf(`
server {
	listen 80;
	server_name %s;

	root %s;

	location / {
		# try to serve file directly, fallback to app.php
		try_files $uri /app.php$is_args$args;
	}

	location ~ ^/app\.php(/|$) {
		fastcgi_pass unix:/var/run/php/php7.0-fpm.sock;
		fastcgi_split_path_info ^(.+\.php)(/.*)$;
		include fastcgi_params;
		fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
		# Prevents URIs that include the front controller. This will 404:
		# http://domain.tld/app.php/some-path
		# Remove the internal directive to allow URIs like this
		internal;
	}

	# Browser caching
	location ~*  \.(jpg|jpeg|png|gif|ico|woff)$ {
		expires 365d;
		try_files $uri /app.php$is_args$args;
	}

	location ~*  \.(css|js)$ {
		expires 30d;
	}

	client_max_body_size 10M;
}`, c.ServerName, c.Root)

	return configFile
}
