@{
    ok     = $true
    before = @{ value = 'bad' }
    after  = @{ value = 'good' }
} | ConvertTo-Json -Compress
