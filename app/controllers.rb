Wallpapers.controllers  do
  get :index do
    @images = get_files
    erb :index, :locals => {}
  end

  get '/image/:id' do
    @image = get_file params[:id]
    redirect @image.public_url
  end


  get '/thumbnail/:id' do

    # TODO: set content-type
    begin
      stream = File.open "tmp/thumb_#{params[:id]}"
    rescue
      @image = get_file params[:id]
      thumbnail = MiniMagick::Image.read(@image.body)
      thumbnail.resize "300x200"
      thumbnail.write "tmp/thumb_#{params[:id]}"
      stream = thumbnail.to_blob
    end

    return stream
  end
end
