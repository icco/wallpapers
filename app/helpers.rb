Wallpapers.helpers do
  def storage_connection
    return Fog::Storage.new({
      :provider                         => 'Google',
      :google_storage_access_key_id     => GOOGLE_KEY,
      :google_storage_secret_access_key => GOOGLE_SECRET,
    })
  end
end
