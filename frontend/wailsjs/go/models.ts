export namespace domain {
	
	export class ArmourStats {
	    Armour: number;
	    DefSkill: number;
	    Shield: number;
	    Sound: string;
	
	    static createFrom(source: any = {}) {
	        return new ArmourStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Armour = source["Armour"];
	        this.DefSkill = source["DefSkill"];
	        this.Shield = source["Shield"];
	        this.Sound = source["Sound"];
	    }
	}
	export class Colour {
	    R: number;
	    G: number;
	    B: number;
	
	    static createFrom(source: any = {}) {
	        return new Colour(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.R = source["R"];
	        this.G = source["G"];
	        this.B = source["B"];
	    }
	}
	export class CostStats {
	    Turns: number;
	    Cost: number;
	    Upkeep: number;
	    WeaponUpgrade: number;
	    ArmourUpgrade: number;
	    Custom: number;
	
	    static createFrom(source: any = {}) {
	        return new CostStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Turns = source["Turns"];
	        this.Cost = source["Cost"];
	        this.Upkeep = source["Upkeep"];
	        this.WeaponUpgrade = source["WeaponUpgrade"];
	        this.ArmourUpgrade = source["ArmourUpgrade"];
	        this.Custom = source["Custom"];
	    }
	}
	export class Faction {
	    Name: string;
	    Culture: string;
	    PrimaryColour: Colour;
	    SecondaryColour: Colour;
	
	    static createFrom(source: any = {}) {
	        return new Faction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Culture = source["Culture"];
	        this.PrimaryColour = this.convertValues(source["PrimaryColour"], Colour);
	        this.SecondaryColour = this.convertValues(source["SecondaryColour"], Colour);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MentalStats {
	    Morale: number;
	    Discipline: string;
	    Training: string;
	
	    static createFrom(source: any = {}) {
	        return new MentalStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Morale = source["Morale"];
	        this.Discipline = source["Discipline"];
	        this.Training = source["Training"];
	    }
	}
	export class SoldierDef {
	    Model: string;
	    Count: number;
	    Extras: number;
	    Mass: number;
	
	    static createFrom(source: any = {}) {
	        return new SoldierDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Model = source["Model"];
	        this.Count = source["Count"];
	        this.Extras = source["Extras"];
	        this.Mass = source["Mass"];
	    }
	}
	export class WeaponStats {
	    Attack: number;
	    ChargeBonus: number;
	    Missile: string;
	    Range: number;
	    Ammo: number;
	    WeaponType: string;
	    TechType: string;
	    DamageType: string;
	    SoundType: string;
	    MinDelay: number;
	    Factor: number;
	
	    static createFrom(source: any = {}) {
	        return new WeaponStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Attack = source["Attack"];
	        this.ChargeBonus = source["ChargeBonus"];
	        this.Missile = source["Missile"];
	        this.Range = source["Range"];
	        this.Ammo = source["Ammo"];
	        this.WeaponType = source["WeaponType"];
	        this.TechType = source["TechType"];
	        this.DamageType = source["DamageType"];
	        this.SoundType = source["SoundType"];
	        this.MinDelay = source["MinDelay"];
	        this.Factor = source["Factor"];
	    }
	}
	export class Unit {
	    Type: string;
	    Dictionary: string;
	    Category: string;
	    Class: string;
	    VoiceType: string;
	    Soldier: SoldierDef;
	    Officers: string[];
	    Mount: string;
	    MountEffect: string;
	    Attributes: string[];
	    Formation: string;
	    StatHealth: number[];
	    StatPri: WeaponStats;
	    StatPriAttr: string[];
	    StatSec: WeaponStats;
	    StatSecAttr: string[];
	    StatPriArmour: ArmourStats;
	    StatSecArmour: ArmourStats;
	    StatHeat: number;
	    StatGround: number[];
	    StatMental: MentalStats;
	    StatChargeDist: number;
	    StatFireDelay: number;
	    StatFood: number[];
	    StatCost: CostStats;
	    Ownership: string[];
	    Name: string;
	    Descr: string;
	    DescrShort: string;
	
	    static createFrom(source: any = {}) {
	        return new Unit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Dictionary = source["Dictionary"];
	        this.Category = source["Category"];
	        this.Class = source["Class"];
	        this.VoiceType = source["VoiceType"];
	        this.Soldier = this.convertValues(source["Soldier"], SoldierDef);
	        this.Officers = source["Officers"];
	        this.Mount = source["Mount"];
	        this.MountEffect = source["MountEffect"];
	        this.Attributes = source["Attributes"];
	        this.Formation = source["Formation"];
	        this.StatHealth = source["StatHealth"];
	        this.StatPri = this.convertValues(source["StatPri"], WeaponStats);
	        this.StatPriAttr = source["StatPriAttr"];
	        this.StatSec = this.convertValues(source["StatSec"], WeaponStats);
	        this.StatSecAttr = source["StatSecAttr"];
	        this.StatPriArmour = this.convertValues(source["StatPriArmour"], ArmourStats);
	        this.StatSecArmour = this.convertValues(source["StatSecArmour"], ArmourStats);
	        this.StatHeat = source["StatHeat"];
	        this.StatGround = source["StatGround"];
	        this.StatMental = this.convertValues(source["StatMental"], MentalStats);
	        this.StatChargeDist = source["StatChargeDist"];
	        this.StatFireDelay = source["StatFireDelay"];
	        this.StatFood = source["StatFood"];
	        this.StatCost = this.convertValues(source["StatCost"], CostStats);
	        this.Ownership = source["Ownership"];
	        this.Name = source["Name"];
	        this.Descr = source["Descr"];
	        this.DescrShort = source["DescrShort"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

