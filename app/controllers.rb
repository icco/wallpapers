Wallpapers.controllers  do
  get :index do
    @images = Storage.get_files
    erb :index, :locals => {}
  end

  get '/image/:id' do
    @image = Storage.get_file params[:id]
    redirect @image.public_url
  end


  get '/thumbnail/:id' do

    # TODO: set content-type
    begin
      image = Storage.get_thumb params[:id]

      if image.nil?
        stream = File.open "tmp/thumb_#{params[:id]}"
      end

      redirect "http://placehold.it/300x200" if Padrino.env == :development
      redirect image.public_url
    rescue
      @image = Storage.get_file params[:id]
      thumbnail = MiniMagick::Image.read(@image.body)
      thumbnail.combine_options do |c|
          c.quality "60"
          c.resize "300x200"
      end
      thumbnail.write "tmp/thumb_#{params[:id]}"

      Storage.thumb_dir.files.create(
        :key    => params[:id],
        :body   => File.open("tmp/thumb_#{params[:id]}"),
        :public => true,
      )

      return thumbnail.to_blob
    end
  end
end
