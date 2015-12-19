#! /bin/bash
source ~/.rvm/environments/ruby-2.2.3

git pull
bundle update
git ci Gemfile* -m 'bundle update'
git st
rake push
rake generate_thumbs
git push
