From ruby:2.6.0

WORKDIR /opt
COPY . .

ENV PORT 8080
ENV RACK_ENV production

RUN bundle install --system --without=test development

CMD bundle exec thin -R config.ru start -p $PORT
EXPOSE 8080
