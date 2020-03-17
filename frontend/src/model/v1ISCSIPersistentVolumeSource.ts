/**
 * Kubernetes
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: v1.15.5
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { V1SecretReference } from './v1SecretReference';

/**
* ISCSIPersistentVolumeSource represents an ISCSI disk. ISCSI volumes can only be mounted as read/write once. ISCSI volumes support ownership management and SELinux relabeling.
*/
export class V1ISCSIPersistentVolumeSource {
    /**
    * whether support iSCSI Discovery CHAP authentication
    */
    'chapAuthDiscovery'?: boolean;
    /**
    * whether support iSCSI Session CHAP authentication
    */
    'chapAuthSession'?: boolean;
    /**
    * Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi
    */
    'fsType'?: string;
    /**
    * Custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection.
    */
    'initiatorName'?: string;
    /**
    * Target iSCSI Qualified Name.
    */
    'iqn': string;
    /**
    * iSCSI Interface Name that uses an iSCSI transport. Defaults to \'default\' (tcp).
    */
    'iscsiInterface'?: string;
    /**
    * iSCSI Target Lun number.
    */
    'lun': number;
    /**
    * iSCSI Target Portal List. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).
    */
    'portals'?: Array<string>;
    /**
    * ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false.
    */
    'readOnly'?: boolean;
    'secretRef'?: V1SecretReference;
    /**
    * iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).
    */
    'targetPortal': string;

    static discriminator: string | undefined = undefined;

    static attributeTypeMap: Array<{name: string, baseName: string, type: string}> = [
        {
            "name": "chapAuthDiscovery",
            "baseName": "chapAuthDiscovery",
            "type": "boolean"
        },
        {
            "name": "chapAuthSession",
            "baseName": "chapAuthSession",
            "type": "boolean"
        },
        {
            "name": "fsType",
            "baseName": "fsType",
            "type": "string"
        },
        {
            "name": "initiatorName",
            "baseName": "initiatorName",
            "type": "string"
        },
        {
            "name": "iqn",
            "baseName": "iqn",
            "type": "string"
        },
        {
            "name": "iscsiInterface",
            "baseName": "iscsiInterface",
            "type": "string"
        },
        {
            "name": "lun",
            "baseName": "lun",
            "type": "number"
        },
        {
            "name": "portals",
            "baseName": "portals",
            "type": "Array<string>"
        },
        {
            "name": "readOnly",
            "baseName": "readOnly",
            "type": "boolean"
        },
        {
            "name": "secretRef",
            "baseName": "secretRef",
            "type": "V1SecretReference"
        },
        {
            "name": "targetPortal",
            "baseName": "targetPortal",
            "type": "string"
        }    ];

    static getAttributeTypeMap() {
        return V1ISCSIPersistentVolumeSource.attributeTypeMap;
    }
}
