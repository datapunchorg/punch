<configuration>
      {{- $metastore_uris :=  regexp.Replace "(.*)" "thrift://$1:9083" .Env.HOSTNAME  }}
      {{- if index .Env "HIVE_METASTORE_URIS"  }}
      {{-   $metastore_uris :=  .Env.HIVE_METASTORE_URIS  }}
      {{- end }}
      {{- if eq .Env.HIVE_DB_EXTERNAL "true" }}
        <property>
          <name>javax.jdo.option.ConnectionURL</name>
          <value>jdbc:postgresql://{{ .Env.HIVE_DB_HOST }}/{{ .Env.HIVE_DB_NAME }}</value>
        </property>
        <property>
          <name>javax.jdo.option.ConnectionDriverName</name>
          <value>org.postgresql.Driver</value>
        </property>
        <property>
          <name>javax.jdo.option.ConnectionUserName</name>
          <value>{{ .Env.HIVE_DB_USER }}</value>
        </property>
        <property>
          <name> javax.jdo.option.ConnectionPassword</name>
          <value>{{ .Env.HIVE_DB_PASS }}</value>
        </property>
      {{- end }}
        <property>
          <name>metastore.expression.proxy</name>
          <value>org.apache.hadoop.hive.metastore.DefaultPartitionExpressionProxy</value>
        </property>
        <property>
          <name>metastore.task.threads.always</name>
          <value>org.apache.hadoop.hive.metastore.events.EventCleanerTask,org.apache.hadoop.hive.metastore.MaterializationsCacheCleanerTask</value>
        </property>
        <property>
          <name>datanucleus.autoCreateSchema</name>
          <value>false</value>
        </property>
        <property>
          <name>hive.metastore.uris</name>
          <value>{{ $metastore_uris }}</value>
        </property>
      {{- if not (index .Env  "HIVE_WAREHOUSE_DIR")  }}
        <property>
          <name>hive.metastore.warehouse.dir</name>
          <value>file:///tmp/</value>
        </property>
      {{- end }}
     {{- if (index .Env "HIVE_CONF_PARAMS")  }}
        {{- $conf_list := .Env.HIVE_CONF_PARAMS | strings.Split ";" }}
        {{- range $parameter := $conf_list}}
            {{- $key := regexp.Replace "(.*):" "$1"  $parameter }}
            {{- $value := regexp.Replace ".*:(.*)" "$1"  $parameter }}
        <property>
          <name>{{ $key }}</name>
          <value>{{ $value }}</value>
        </property>
       {{- end }}
     {{- end }}

</configuration>