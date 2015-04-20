require "rubygems"
require "bundler"
Bundler.require(:default, ENV["RACK_ENV"] || :development)
require "./lib/storage.rb"

configure do
  FORCE_PROD = true
end

get "/" do
  @images = Storage.get_files(FORCE_PROD)
  erb :index
end

get "/image/:id" do
  @image = Storage.get_file(params[:id], FORCE_PROD)

  if @image.nil?
    404
  else
    if @image.file_url
      redirect @image.file_url
    else
      403
    end
  end
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
