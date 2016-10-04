require "rubygems"
require "bundler"
Bundler.require(:default, ENV["RACK_ENV"] || :development)
require "./lib/storage.rb"

configure do
  FORCE_PROD = true
end

get "/" do
  erb :index
end

get "/all.json" do
  @images = Storage.get_files(FORCE_PROD).map do |i|
    {
      image: i.file_url,
      key: i.key,
      thumbnail: i.thumb_url,
      cdn: i.cdn_url,
      etag: i.etag,
    }
  end

  content_type :json
  @images.to_json
end

get "/403" do
  403
end

get "/404" do
  404
end

error 400..510 do
  @code = response.status
  erb :error
end
