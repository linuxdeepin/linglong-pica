## ll-pica init 常见问题
TODO

## ll-pica convert 常见问题
### deb包的/opt数据未拷贝到玲珑包中
用户需确认/opt/apps/<appid>路径中的appid是否与appid.yaml配置文件中的appid是否一致，不一致时，无法拷贝/opt下数据。商店提供包时，尽量包名与appid保持一致。