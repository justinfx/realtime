<?php
class RT {
	
	var $domain = "";
	var $api = "/api/publish";
	var $status = 0;
	
	function __construct($domain) {
		if($domain) $this->domain = $domain;
	}
	
	function publish($channel,$msg) {
	
		$ch = curl_init();
		curl_setopt($ch, CURLOPT_URL,$this->domain.$this->api);
		curl_setopt($ch, CURLOPT_RETURNTRANSFER,1);
		curl_setopt($ch, CURLOPT_POST, 1);
		
		$json = json_encode(array(
			"type"=>"command",
			"channel"=>$channel,
			"identity"=>"sean",
			"data"=>array(
				"command"=>"onTest"
			)
		));
		
		// set post
		curl_setopt($ch,CURLOPT_POSTFIELDS,$json);
		//exce
		$ret = curl_exec($ch);
		
		// set the http status
		$this->status = curl_getinfo($ch,CURLINFO_HTTP_CODE);
		return $ret;
	}
}
?>