Wallpapers.helpers do
  def storage_connection
    directory_name = "iccowalls"

    if Padrino.env == :development
      credentials = {
        :provider   => "Local",
        :local_root => "/tmp/",
      }
    else
      credentials = {
        :provider                         => 'Google',
        :google_storage_access_key_id     => GOOGLE_KEY,
        :google_storage_secret_access_key => GOOGLE_SECRET,
      }
    end

    connection = Fog::Storage.new(credentials)

    directory = connection.directories.get(directory_name)

    if directory.nil?
      directory = connection.directories.create(
        :key    => directory_name,
        :public => true
      )
    end

    return directory
  end

  def get_files
    return storage_connection.files
  end

  def get_file filename
    return get_files.get(filename)
  end
end
