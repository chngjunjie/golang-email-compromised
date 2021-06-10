#nginx/Dockerfile
FROM nginx:latest
EXPOSE 80
COPY ./templates/*.html /usr/src/html/
COPY ./static/* /usr/src/static/
COPY nginx.conf /etc/nginx/nginx.conf
COPY mime.types /etc/nginx/mime.types