#!/bin/sh

host=$1
port=$2

# Get HTTP request and parse header for URL path
while read -t 5 line
do
    if echo $line | grep -q " HTTP/1.1" ; then
        path=`echo $line | awk '{print $2;}'`
        break
    fi
done

# Invalid request.
[ -n "$path" ] || exit 0

ip=`getent hosts $host | awk '{print $1;}'`

if [ "$ip" = "" ] ; then
    echo "HTTP/1.1 200"
    echo "Content-type: text/html"
    echo "Cache-Control: no-cache, no-store, must-revalidate"
    echo
    echo "<h2>Serivce $host not up</h2>"
    echo
else
    # HTTP/1.1 states that client rewrite method to GET for code 301/302,
    # even when original request is POST. 307 keeps original method. 
    echo "HTTP/1.1 307 Temporary Redirect"
    echo "Cache-Control: no-cache, no-store, must-revalidate"
    echo "Location: http://$ip:$port$path"
    echo
fi
