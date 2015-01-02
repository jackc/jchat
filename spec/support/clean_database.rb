def clean_database
  %i[messages channels password_resets users].each do |t|
    DB[t].delete
  end
end
