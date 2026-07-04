# FortByte Demo
# Run: vhs demo/fortbyte.vhs

Output "demo/fortbyte.mp4"

# Canvas
Set Width 960
Set Height 540

# Typography
Set FontFamily "Menlo"
Set FontSize 18

# Appearance
Set Theme "Catppuccin Mocha"
Set Padding 24

# Animation
Set TypingSpeed 75ms
Set CursorBlink false

# --------------------------------------------------
# 2. Initialize vault
# --------------------------------------------------
Type "./fort init"
Enter
Sleep 750ms

Hide
Type "s3cret!Pass"
Enter
Sleep 300ms

Type "s3cret!Pass"
Enter
Show

Sleep 2s

# --------------------------------------------------
# 3. Create project
# --------------------------------------------------
Type `./fort project add webapp --desc "My web application" --url "https://example.com"`
Enter
Sleep 1500ms

# --------------------------------------------------
# 4. Create environment
# --------------------------------------------------
Type "./fort env add production --project webapp"
Enter
Sleep 1500ms

# --------------------------------------------------
# 5. Add database secret
# --------------------------------------------------
Type "./fort secret add DATABASE_URL --project webapp --env production"
Enter
Sleep 300ms
Hide
Type "postgres://admin:s3cret@localhost:5432/mydb"
Enter
Show
Sleep 300ms
Type "https://db.example.com"
Enter
Sleep 300ms
Type "Primary database"
Enter
Sleep 1500ms

# --------------------------------------------------
# 6. Add API key
# --------------------------------------------------
Type "./fort secret add API_KEY --project webapp --env production"
Enter
Sleep 300ms
Hide
Type "ak_live_abcdef1234567890"
Enter
Show
Sleep 300ms
Enter
Sleep 300ms
Type "Stripe API key"
Enter
Sleep 1500ms

# --------------------------------------------------
# 7. List secrets
# --------------------------------------------------
Type "./fort list"
Enter
Sleep 2s

# --------------------------------------------------
# 8. Reveal value
# --------------------------------------------------
Type "./fort secret reveal DATABASE_URL --project webapp --env production"
Enter
Sleep 2s

# --------------------------------------------------
# 9. Export
# --------------------------------------------------
Type "./fort export /tmp/fortbyte-backup.json --project webapp"
Enter
Sleep 1500ms
Type "yes"
Enter
Sleep 500ms

# --------------------------------------------------
# 10. Import
# --------------------------------------------------
Type "./fort import /tmp/fortbyte-backup.json --project webapp"
Enter
Sleep 2s