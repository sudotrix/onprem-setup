for server in `cat server.txt`;  
do  
    sshpass -p "Lowes@123" ssh-copy-id -i ~/.ssh/id_rsa.pub s0998rpx@$server  
done
