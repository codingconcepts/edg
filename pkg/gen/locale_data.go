package gen

type localeData struct {
	FirstNames  []string
	LastNames   []string
	Cities      []string
	Streets     []string
	PhoneFmt    string
	ZipFmt      string
	AddressFmt  string
	NameOrder   string // "western" (first last) or "eastern" (last first)
}

var locales = map[string]*localeData{
	"en_US": {
		FirstNames: []string{
			"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda",
			"David", "Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
			"Thomas", "Sarah", "Christopher", "Karen", "Charles", "Lisa", "Daniel", "Nancy",
			"Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra", "Donald", "Ashley",
			"Steven", "Dorothy", "Andrew", "Kimberly", "Paul", "Emily", "Joshua", "Donna",
		},
		LastNames: []string{
			"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
			"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson",
			"Thomas", "Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson",
			"White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
		},
		Cities: []string{
			"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia",
			"San Antonio", "San Diego", "Dallas", "Austin", "Jacksonville", "Fort Worth",
			"Columbus", "Charlotte", "Indianapolis", "San Francisco", "Seattle", "Denver",
			"Nashville", "Portland", "Memphis", "Louisville", "Baltimore", "Milwaukee",
		},
		Streets: []string{
			"Main Street", "Oak Avenue", "Cedar Lane", "Elm Street", "Maple Drive",
			"Washington Boulevard", "Park Avenue", "Lake Street", "Hill Road", "River Road",
			"Sunset Boulevard", "Pine Street", "Broadway", "Market Street", "Church Street",
		},
		PhoneFmt:   "[2-9][0-9]{2}-[2-9][0-9]{2}-[0-9]{4}",
		ZipFmt:     "[0-9]{5}",
		AddressFmt: "western",
		NameOrder:  "western",
	},

	"ja_JP": {
		FirstNames: []string{
			"太郎", "花子", "一郎", "美咲", "健太", "陽子", "翔太", "さくら",
			"大輔", "愛", "拓也", "由美", "直樹", "恵", "雄太", "麻衣",
			"和也", "裕子", "亮", "真由美", "隆", "智子", "誠", "明美",
			"浩", "京子", "哲也", "幸子", "剛", "典子", "学", "久美子",
			"悠斗", "結衣", "蓮", "凛", "湊", "陽菜", "悠真", "葵",
		},
		LastNames: []string{
			"佐藤", "鈴木", "高橋", "田中", "伊藤", "渡辺", "山本", "中村",
			"小林", "加藤", "吉田", "山田", "佐々木", "松本", "井上", "木村",
			"林", "斎藤", "清水", "山崎", "森", "池田", "橋本", "阿部",
			"石川", "山下", "中島", "石井", "小川", "前田",
		},
		Cities: []string{
			"東京", "大阪", "横浜", "名古屋", "札幌", "福岡", "神戸", "京都",
			"川崎", "さいたま", "広島", "仙台", "千葉", "北九州", "堺", "新潟",
			"浜松", "熊本", "相模原", "岡山", "静岡", "船橋", "鹿児島", "八王子",
		},
		Streets: []string{
			"中央通り", "表参道", "御堂筋", "国道一号線", "明治通り",
			"青山通り", "靖国通り", "甲州街道", "環七通り", "環八通り",
			"外堀通り", "内堀通り", "昭和通り", "日比谷通り", "白山通り",
		},
		PhoneFmt:   "0[0-9]{1,3}-[0-9]{2,4}-[0-9]{4}",
		ZipFmt:     "[0-9]{3}-[0-9]{4}",
		AddressFmt: "eastern",
		NameOrder:  "eastern",
	},

	"de_DE": {
		FirstNames: []string{
			"Hans", "Anna", "Klaus", "Maria", "Wolfgang", "Ursula", "Jürgen", "Helga",
			"Dieter", "Monika", "Peter", "Petra", "Thomas", "Sabine", "Michael", "Gabriele",
			"Andreas", "Christine", "Stefan", "Claudia", "Markus", "Susanne", "Frank", "Birgit",
			"Matthias", "Andrea", "Martin", "Stefanie", "Bernd", "Karin", "Uwe", "Renate",
			"Lukas", "Sophie", "Finn", "Mia", "Leon", "Emma", "Elias", "Hannah",
		},
		LastNames: []string{
			"Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker",
			"Schulz", "Hoffmann", "Schäfer", "Koch", "Bauer", "Richter", "Klein", "Wolf",
			"Schröder", "Neumann", "Schwarz", "Zimmermann", "Braun", "Krüger", "Hofmann",
			"Hartmann", "Lange", "Schmitt", "Werner", "Schmitz", "Krause", "Meier",
		},
		Cities: []string{
			"Berlin", "Hamburg", "München", "Köln", "Frankfurt", "Stuttgart", "Düsseldorf",
			"Leipzig", "Dortmund", "Essen", "Bremen", "Dresden", "Hannover", "Nürnberg",
			"Duisburg", "Bochum", "Wuppertal", "Bielefeld", "Bonn", "Münster",
		},
		Streets: []string{
			"Hauptstraße", "Bahnhofstraße", "Schulstraße", "Gartenstraße", "Dorfstraße",
			"Bergstraße", "Birkenweg", "Lindenstraße", "Kirchstraße", "Waldstraße",
			"Ringstraße", "Schillerstraße", "Goethestraße", "Mozartstraße", "Friedrichstraße",
		},
		PhoneFmt:   "\\+49 [0-9]{3} [0-9]{7,8}",
		ZipFmt:     "[0-9]{5}",
		AddressFmt: "western",
		NameOrder:  "western",
	},

	"fr_FR": {
		FirstNames: []string{
			"Jean", "Marie", "Pierre", "Françoise", "Michel", "Monique", "André", "Nicole",
			"Philippe", "Jacqueline", "Jacques", "Sylvie", "Bernard", "Nathalie", "Alain", "Isabelle",
			"François", "Catherine", "Patrick", "Véronique", "Daniel", "Christine", "Christophe", "Sophie",
			"Laurent", "Anne", "Thierry", "Martine", "Nicolas", "Sandrine", "Éric", "Valérie",
			"Lucas", "Emma", "Hugo", "Jade", "Louis", "Louise", "Raphaël", "Alice",
		},
		LastNames: []string{
			"Martin", "Bernard", "Dubois", "Thomas", "Robert", "Richard", "Petit", "Durand",
			"Leroy", "Moreau", "Simon", "Laurent", "Lefebvre", "Michel", "Garcia", "David",
			"Bertrand", "Roux", "Vincent", "Fournier", "Morel", "Girard", "André", "Mercier",
			"Dupont", "Lambert", "Bonnet", "François", "Martinez", "Legrand",
		},
		Cities: []string{
			"Paris", "Marseille", "Lyon", "Toulouse", "Nice", "Nantes", "Strasbourg",
			"Montpellier", "Bordeaux", "Lille", "Rennes", "Reims", "Toulon", "Le Havre",
			"Saint-Étienne", "Grenoble", "Dijon", "Angers", "Nîmes", "Villeurbanne",
		},
		Streets: []string{
			"Rue de la Paix", "Avenue des Champs-Élysées", "Boulevard Saint-Germain",
			"Rue du Faubourg Saint-Honoré", "Avenue Montaigne", "Rue de Rivoli",
			"Boulevard Haussmann", "Rue de la République", "Avenue Victor Hugo",
			"Rue Nationale", "Place de la Liberté", "Rue du Commerce",
			"Boulevard Voltaire", "Rue des Fleurs", "Avenue Jean Jaurès",
		},
		PhoneFmt:   "\\+33 [1-9] [0-9]{2} [0-9]{2} [0-9]{2} [0-9]{2}",
		ZipFmt:     "[0-9]{5}",
		AddressFmt: "western",
		NameOrder:  "western",
	},

	"es_ES": {
		FirstNames: []string{
			"Antonio", "María", "José", "Carmen", "Manuel", "Ana", "Francisco", "Laura",
			"David", "Marta", "Juan", "Cristina", "Carlos", "Lucía", "Javier", "Elena",
			"Miguel", "Isabel", "Ángel", "Rosa", "Pedro", "Pilar", "Rafael", "Teresa",
			"Fernando", "Dolores", "Alejandro", "Raquel", "Pablo", "Sara", "Sergio", "Paula",
			"Hugo", "Sofía", "Martín", "Valentina", "Lucas", "Daniela", "Leo", "Alba",
		},
		LastNames: []string{
			"García", "Rodríguez", "Martínez", "López", "González", "Hernández", "Pérez",
			"Sánchez", "Ramírez", "Torres", "Flores", "Rivera", "Gómez", "Díaz", "Cruz",
			"Morales", "Reyes", "Gutiérrez", "Ortiz", "Ruiz", "Álvarez", "Muñoz", "Romero",
			"Jiménez", "Navarro", "Domínguez", "Moreno", "Molina", "Iglesias", "Suárez",
		},
		Cities: []string{
			"Madrid", "Barcelona", "Valencia", "Sevilla", "Zaragoza", "Málaga", "Murcia",
			"Palma", "Las Palmas", "Bilbao", "Alicante", "Córdoba", "Valladolid", "Vigo",
			"Gijón", "Hospitalet", "Vitoria", "Granada", "Elche", "Oviedo",
		},
		Streets: []string{
			"Calle Mayor", "Gran Vía", "Paseo de la Castellana", "Avenida de la Constitución",
			"Calle Real", "Paseo del Prado", "Calle de Alcalá", "Rambla de Catalunya",
			"Avenida de América", "Calle de Serrano", "Calle del Carmen", "Paseo de Gracia",
			"Avenida de Andalucía", "Calle de la Paz", "Calle San Fernando",
		},
		PhoneFmt:   "\\+34 [6-9][0-9]{2} [0-9]{3} [0-9]{3}",
		ZipFmt:     "[0-9]{5}",
		AddressFmt: "western",
		NameOrder:  "western",
	},

	"pt_BR": {
		FirstNames: []string{
			"João", "Maria", "José", "Ana", "Pedro", "Francisca", "Paulo", "Adriana",
			"Carlos", "Juliana", "Lucas", "Mariana", "Mateus", "Fernanda", "Rafael", "Patrícia",
			"Gabriel", "Aline", "Bruno", "Camila", "Felipe", "Amanda", "Gustavo", "Bruna",
			"Thiago", "Letícia", "Leonardo", "Larissa", "Daniel", "Beatriz", "Rodrigo", "Carolina",
			"Miguel", "Alice", "Arthur", "Sophia", "Heitor", "Helena", "Bernardo", "Valentina",
		},
		LastNames: []string{
			"Silva", "Santos", "Oliveira", "Souza", "Rodrigues", "Ferreira", "Alves", "Pereira",
			"Lima", "Gomes", "Costa", "Ribeiro", "Martins", "Carvalho", "Almeida", "Lopes",
			"Soares", "Fernandes", "Vieira", "Barbosa", "Rocha", "Dias", "Nascimento",
			"Andrade", "Moreira", "Nunes", "Marques", "Machado", "Mendes", "Freitas",
		},
		Cities: []string{
			"São Paulo", "Rio de Janeiro", "Brasília", "Salvador", "Fortaleza", "Belo Horizonte",
			"Manaus", "Curitiba", "Recife", "Goiânia", "Porto Alegre", "Belém", "Guarulhos",
			"Campinas", "São Luís", "Maceió", "Campo Grande", "Teresina", "João Pessoa", "Natal",
		},
		Streets: []string{
			"Rua das Flores", "Avenida Paulista", "Rua Augusta", "Avenida Brasil",
			"Rua XV de Novembro", "Avenida Atlântica", "Rua da Consolação",
			"Avenida Rio Branco", "Rua Oscar Freire", "Rua da Liberdade",
			"Avenida Copacabana", "Rua São Paulo", "Rua Direita", "Avenida Ipiranga",
			"Rua da Paz",
		},
		PhoneFmt:   "\\+55 [0-9]{2} 9[0-9]{4}-[0-9]{4}",
		ZipFmt:     "[0-9]{5}-[0-9]{3}",
		AddressFmt: "western",
		NameOrder:  "western",
	},

	"zh_CN": {
		FirstNames: []string{
			"伟", "芳", "娜", "秀英", "敏", "静", "丽", "强", "磊", "洋",
			"勇", "艳", "杰", "娟", "涛", "明", "超", "秀兰", "霞", "平",
			"刚", "桂英", "文", "华", "军", "慧", "玲", "建华", "建国", "建军",
			"志强", "美兰", "晓明", "小红", "大伟", "海燕", "建平", "丽丽", "天宇", "子涵",
		},
		LastNames: []string{
			"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "吴", "周",
			"徐", "孙", "马", "胡", "朱", "郭", "何", "林", "罗", "高",
			"郑", "梁", "谢", "宋", "唐", "许", "邓", "韩", "冯", "曹",
		},
		Cities: []string{
			"北京", "上海", "广州", "深圳", "天津", "重庆", "成都", "武汉",
			"杭州", "南京", "苏州", "西安", "长沙", "郑州", "青岛", "大连",
			"沈阳", "厦门", "无锡", "济南", "合肥", "福州", "昆明", "哈尔滨",
		},
		Streets: []string{
			"长安街", "南京路", "淮海路", "中山路", "解放路",
			"人民路", "建设路", "和平路", "新华路", "胜利路",
			"光明路", "幸福路", "团结路", "文化路", "科技路",
		},
		PhoneFmt:   "1[3-9][0-9]{9}",
		ZipFmt:     "[0-9]{6}",
		AddressFmt: "eastern",
		NameOrder:  "eastern",
	},

	"ko_KR": {
		FirstNames: []string{
			"민준", "서윤", "서준", "지우", "도윤", "서연", "예준", "하윤",
			"시우", "하은", "주원", "지유", "하준", "지윤", "지호", "은서",
			"준서", "수아", "건우", "지아", "현우", "채원", "민서", "소윤",
			"우진", "예은", "선우", "시은", "준우", "하린", "정우", "유나",
			"지훈", "미나", "성민", "수진", "영호", "은지", "재현", "혜진",
		},
		LastNames: []string{
			"김", "이", "박", "최", "정", "강", "조", "윤", "장", "임",
			"한", "오", "서", "신", "권", "황", "안", "송", "류", "전",
			"홍", "고", "문", "양", "손", "배", "백", "허", "유", "남",
		},
		Cities: []string{
			"서울", "부산", "인천", "대구", "대전", "광주", "울산", "수원",
			"성남", "고양", "용인", "창원", "청주", "전주", "천안", "화성",
			"남양주", "제주", "평택", "포항", "김해", "안산", "시흥", "파주",
		},
		Streets: []string{
			"세종대로", "테헤란로", "강남대로", "올림픽대로", "종로",
			"을지로", "충무로", "명동길", "압구정로", "삼성로",
			"반포대로", "영동대로", "도산대로", "언주로", "논현로",
		},
		PhoneFmt:   "010-[0-9]{4}-[0-9]{4}",
		ZipFmt:     "[0-9]{5}",
		AddressFmt: "eastern",
		NameOrder:  "eastern",
	},
}
