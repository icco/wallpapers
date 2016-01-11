#! /bin/bash

RUBY=~/.rvm/environments/ruby-2.3.0

if [[ ! -f $RUBY ]] ; then
  echo "File $RUBY is not there, aborting."
  exit
fi

source $RUBY

git pull
bundle update
git ci Gemfile* -m 'bundle update'
git st
rake push
git push
